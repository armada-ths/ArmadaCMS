package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func loungeTestDatabase(t *testing.T) (*gorm.DB, uint, uint, uint) {
	t.Helper()
	dsn := os.Getenv("LOUNGE_TEST_DSN")
	if dsn == "" {
		t.Skip("LOUNGE_TEST_DSN is required for isolated database tests")
	}
	if !strings.Contains(dsn, "dbname=lounge_plan_test") {
		t.Fatal("refusing to test against a non-isolated database")
	}
	connection, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	old := db.DB
	db.DB = connection
	t.Cleanup(func() {
		db.DB = old
		sql, _ := connection.DB()
		if sql != nil {
			_ = sql.Close()
		}
	})
	if err := connection.Exec("TRUNCATE lounge_order_lines, lounge_orders, lounge_items, lounge_categories, fair_date_configs, exhibitors CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := connection.Exec("UPDATE lounge_catalogue_state SET revision = 0 WHERE id = 1").Error; err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOUNGE_ORDER_SUBMISSION_SECRET", "live-service")
	t.Setenv("LOUNGE_ORDER_TEST_SUBMISSION_SECRET", "test-service")
	category := models.LoungeCategory{Name: "Drinks"}
	if err := connection.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	max := int64(3)
	item := models.LoungeItem{Name: "Coffee", CategoryID: category.ID, MaxPerOrder: &max}
	if err := connection.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	if err := connection.Create(&models.FairDateConfig{Description: "Test", FairDays: "2099-11-17,2099-11-18"}).Error; err != nil {
		t.Fatal(err)
	}
	var company uint
	if err := connection.Raw("INSERT INTO exhibitors(name, type, tier, fair_location, climate_compensation, flyer) VALUES ('Gold AB', 'Test', 'Gold', 'A1', false, '') RETURNING id").Scan(&company).Error; err != nil {
		t.Fatal(err)
	}
	return connection, company, category.ID, item.ID
}

func loungeRequestTest(handler http.HandlerFunc, method, path string, body any, credential string, id uint) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	if id != 0 {
		request = mux.SetURLVars(request, map[string]string{"id": fmt.Sprint(id)})
	}
	response := httptest.NewRecorder()
	handler(response, request)
	return response
}

func loungeInput(company, item uint, key string) map[string]any {
	return map[string]any{"idempotencyKey": key, "exhibitorId": company, "items": []map[string]any{{"id": fmt.Sprint(item), "quantity": 2}}}
}

func TestLoungeOpeningBoundaries(t *testing.T) {
	zone, _ := time.LoadLocation("Europe/Stockholm")
	for _, example := range []struct {
		time string
		open bool
	}{
		{"2026-11-16T12:00:00", true}, {"2026-11-17T15:30:00", true}, {"2026-11-17T15:30:01", false},
		{"2026-11-18T09:59:59", false}, {"2026-11-18T10:00:00", true}, {"2026-11-18T14:30:00", true},
		{"2026-11-18T14:30:01", false}, {"2026-11-19T10:00:00", false},
	} {
		now, _ := time.ParseInLocation("2006-01-02T15:04:05", example.time, zone)
		if loungeOpen("2026-11-17,2026-11-18", now) != example.open {
			t.Fatalf("incorrect opening at %s", example.time)
		}
	}
	if loungeOpen("invalid,2026-11-18", time.Now()) || loungeOpen("2026-11-18,2026-11-17", time.Now()) {
		t.Fatal("invalid dates accepted")
	}
}

func TestLoungePersistenceIdempotencyAndHistory(t *testing.T) {
	connection, company, category, item := loungeTestDatabase(t)
	input := loungeInput(company, item, "00000000-0000-4000-8000-000000000001")
	response := loungeRequestTest(SubmitLoungeOrder, "POST", "/lounge-orders/submit", input, "test-service", 0)
	if response.Code != 201 {
		t.Fatalf("submit: %d %s", response.Code, response.Body)
	}
	var order models.LoungeOrder
	if err := json.Unmarshal(response.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	if len(order.Lines) != 1 || order.Lines[0].ItemName != "Coffee" || !order.IsTest {
		t.Fatalf("bad snapshot: %+v", order)
	}
	deleted := loungeRequestTest(DeleteLoungeCategory, "DELETE", "/lounge-categories/1", nil, "", category)
	if deleted.Code != 204 {
		t.Fatalf("delete: %d %s", deleted.Code, deleted.Body)
	}
	if err := connection.Exec("DELETE FROM exhibitors WHERE id = ?", company).Error; err != nil {
		t.Fatal(err)
	}
	again := loungeRequestTest(SubmitLoungeOrder, "POST", "/lounge-orders/submit", input, "test-service", 0)
	if again.Code != 200 {
		t.Fatalf("idempotent replay after deletion: %d %s", again.Code, again.Body)
	}
	var saved models.LoungeOrder
	_ = json.Unmarshal(again.Body.Bytes(), &saved)
	if saved.ID != order.ID || saved.CompanyName != "Gold AB" || saved.Lines[0].CategoryName != "Drinks" {
		t.Fatal("history changed or duplicate order")
	}
	var count int64
	connection.Model(&models.LoungeItem{}).Count(&count)
	if count != 0 {
		t.Fatal("category deletion left items")
	}
	input["items"] = []map[string]any{{"id": fmt.Sprint(item), "quantity": 1}}
	if changed := loungeRequestTest(SubmitLoungeOrder, "POST", "/lounge-orders/submit", input, "test-service", 0); changed.Code != 400 {
		t.Fatal("accepted different payload with same key")
	}
}

func TestLoungeConcurrentSubmissionAndClaims(t *testing.T) {
	connection, company, _, item := loungeTestDatabase(t)
	input := loungeInput(company, item, "00000000-0000-4000-8000-000000000002")
	var group sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 6)
	for i := 0; i < 6; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			results <- loungeRequestTest(SubmitLoungeOrder, "POST", "/lounge-orders/submit", input, "test-service", 0)
		}()
	}
	group.Wait()
	close(results)
	for response := range results {
		if response.Code != 200 && response.Code != 201 {
			t.Fatalf("concurrent submission: %d %s", response.Code, response.Body)
		}
	}
	var orders []models.LoungeOrder
	connection.Find(&orders)
	if len(orders) != 1 {
		t.Fatal("duplicate orders")
	}
	claim := loungeRequestTest(ClaimLoungeDelivery, "POST", "/claim", nil, "test-service", orders[0].ID)
	var granted struct {
		Claimed bool   `json:"claimed"`
		Token   string `json:"claimToken"`
	}
	_ = json.Unmarshal(claim.Body.Bytes(), &granted)
	if !granted.Claimed || granted.Token == "" {
		t.Fatalf("claim failed: %s", claim.Body)
	}
	second := loungeRequestTest(ClaimLoungeDelivery, "POST", "/claim", nil, "test-service", orders[0].ID)
	if !strings.Contains(second.Body.String(), `"claimed":false`) {
		t.Fatal("duplicate delivery claim")
	}
	live := loungeRequestTest(ClaimLoungeDelivery, "POST", "/claim", nil, "live-service", orders[0].ID)
	if live.Code == 200 {
		t.Fatal("live credential accessed test order")
	}
	report := loungeRequestTest(ReportLoungeDelivery, "POST", "/delivery", map[string]any{"claimToken": granted.Token, "status": "failed", "error": "slack_rejected"}, "test-service", orders[0].ID)
	if report.Code != 200 {
		t.Fatalf("report failed: %s", report.Body)
	}
	var updated models.LoungeOrder
	connection.First(&updated, orders[0].ID)
	if updated.Status != "pending" || updated.DeliveryStatus != "failed" {
		t.Fatal("delivery changed order acceptance")
	}
}

func TestLoungeValidationAndTotals(t *testing.T) {
	connection, company, _, item := loungeTestDatabase(t)
	input := loungeInput(company, item, "00000000-0000-4000-8000-000000000003")
	if unauthorized := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "", 0); unauthorized.Code != 401 {
		t.Fatal("unauthenticated submission accepted")
	}
	if err := connection.Model(&models.LoungeItem{}).Where("id = ?", item).Update("sold_out", true).Error; err != nil {
		t.Fatal(err)
	}
	if failed := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0); failed.Code != 400 || !strings.Contains(failed.Body.String(), "items_changed") {
		t.Fatal("sold-out item accepted")
	}
	connection.Model(&models.LoungeItem{}).Where("id = ?", item).Update("sold_out", false)
	response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "live-service", 0)
	if response.Code != 201 {
		t.Fatalf("live submit: %s", response.Body)
	}
	var order models.LoungeOrder
	_ = json.Unmarshal(response.Body.Bytes(), &order)
	input["idempotencyKey"] = "00000000-0000-4000-8000-000000000004"
	loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0)
	totals := loungeRequestTest(LoungeOrderTotals, "GET", "/totals", nil, "", 0)
	if totals.Code != 200 || !strings.Contains(totals.Body.String(), `"quantity":"2"`) {
		t.Fatalf("test/live totals incorrect: %s", totals.Body)
	}
	cancelled := loungeRequestTest(UpdateLoungeOrder, "PUT", "/order", map[string]string{"status": "cancelled"}, "", order.ID)
	if cancelled.Code != 200 {
		t.Fatalf("status update: %s", cancelled.Body)
	}
	totals = loungeRequestTest(LoungeOrderTotals, "GET", "/totals", nil, "", 0)
	if totals.Body.String() != "[]\n" {
		t.Fatalf("cancelled orders counted: %s", totals.Body)
	}
}

func TestLoungeReorderRollbackAndMoves(t *testing.T) {
	connection, _, category, item := loungeTestDatabase(t)
	other := models.LoungeCategory{Name: "Snacks", SortOrder: 1}
	if err := connection.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	input := map[string]any{"revision": 0, "categories": []map[string]any{{"id": other.ID, "itemIds": []uint{item}}, {"id": category, "itemIds": []uint{}}}}
	response := loungeRequestTest(ReorderLoungeCatalogue, "PUT", "/catalogue", input, "", 0)
	if response.Code != 200 {
		t.Fatalf("reorder: %s", response.Body)
	}
	var moved models.LoungeItem
	connection.First(&moved, item)
	if moved.CategoryID != other.ID || moved.SortOrder != 0 {
		t.Fatal("move did not persist")
	}
	stale := loungeRequestTest(ReorderLoungeCatalogue, "PUT", "/catalogue", input, "", 0)
	if stale.Code != 409 {
		t.Fatal("stale reorder accepted")
	}
	var revision int64
	connection.Raw("SELECT revision FROM lounge_catalogue_state WHERE id=1").Scan(&revision)
	if revision != 1 {
		t.Fatal("failed reorder changed revision")
	}
}

func TestLoungeAuditFailureRollsBackOrder(t *testing.T) {
	connection, company, _, item := loungeTestDatabase(t)
	err := connection.Exec(`CREATE FUNCTION lounge_test_reject_audit() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN IF NEW.resource_type = 'lounge-orders' THEN RAISE EXCEPTION 'test audit failure'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER lounge_test_audit_failure BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION lounge_test_reject_audit();`).Error
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := connection.Exec("DROP TRIGGER lounge_test_audit_failure ON audit_logs; DROP FUNCTION lounge_test_reject_audit()").Error; err != nil {
			t.Error(err)
		}
	})
	response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", loungeInput(company, item, "00000000-0000-4000-8000-000000000005"), "test-service", 0)
	if response.Code != 503 {
		t.Fatalf("audit failure was accepted: %d %s", response.Code, response.Body)
	}
	var count int64
	if err := connection.Model(&models.LoungeOrder{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("failed audit left an order")
	}
	if err := connection.Model(&models.LoungeOrderLine{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("failed audit left order lines")
	}
}

func TestLoungeFiltersMatchTheSameLineAndTotalsIgnorePagination(t *testing.T) {
	connection, company, categoryID, item := loungeTestDatabase(t)
	second := models.LoungeCategory{Name: "Snacks"}
	if err := connection.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	bun := models.LoungeItem{Name: "Bun", CategoryID: second.ID}
	if err := connection.Create(&bun).Error; err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"00000000-0000-4000-8000-000000000006", "00000000-0000-4000-8000-000000000007"} {
		input := loungeInput(company, item, key)
		input["items"] = []map[string]any{{"id": fmt.Sprint(item), "quantity": 2}, {"id": fmt.Sprint(bun.ID), "quantity": 3}}
		response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "live-service", 0)
		if response.Code != 201 {
			t.Fatalf("fixture submission failed: %s", response.Body)
		}
	}
	filter := url.QueryEscape(fmt.Sprintf(`{"categoryId":%d,"itemId":%d}`, categoryID, bun.ID))
	response := loungeRequestTest(ListLoungeOrders, "GET", "/orders?filter="+filter, nil, "", 0)
	if response.Code != 200 || response.Body.String() != "[]\n" {
		t.Fatalf("different lines matched combined filters: %s", response.Body)
	}
	filter = url.QueryEscape(fmt.Sprintf(`{"itemId":%d}`, bun.ID))
	response = loungeRequestTest(ListLoungeOrders, "GET", "/orders?range=%5B0%2C0%5D&filter="+filter, nil, "", 0)
	var records []models.LoungeOrder
	if json.Unmarshal(response.Body.Bytes(), &records) != nil || len(records) != 1 || len(records[0].Lines) != 2 || !strings.HasSuffix(response.Header().Get("Content-Range"), "/2") {
		t.Fatalf("pagination or full-order details incorrect: %s", response.Body)
	}
	response = loungeRequestTest(LoungeOrderTotals, "GET", "/totals?range=%5B0%2C0%5D&filter="+filter, nil, "", 0)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"quantity":"6"`) || strings.Contains(response.Body.String(), "Coffee") {
		t.Fatalf("totals counted a page or unmatched lines: %s", response.Body)
	}
}

func TestLoungeRejectsInvalidSubmissions(t *testing.T) {
	connection, company, _, item := loungeTestDatabase(t)
	for _, quantity := range []any{0, -1, 1.5, 9007199254740992, 4} {
		input := loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
		input["items"] = []map[string]any{{"id": fmt.Sprint(item), "quantity": quantity}}
		response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0)
		if response.Code != 400 {
			t.Fatalf("quantity %v accepted: %d %s", quantity, response.Code, response.Body)
		}
	}
	input := loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
	input["items"] = []map[string]any{{"id": fmt.Sprint(item), "quantity": 1}, {"id": fmt.Sprint(item), "quantity": 1}}
	if response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0); response.Code != 400 {
		t.Fatal("duplicate lines accepted")
	}
	input = loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
	input["items"] = []map[string]any{{"id": "9007199254740991", "quantity": 1}}
	if response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0); response.Code != 400 || !strings.Contains(response.Body.String(), "items_changed") {
		t.Fatal("missing item accepted")
	}
	input = loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
	if err := connection.Model(&models.Exhibitor{}).Where("id = ?", company).Update("tier", "Silver").Error; err != nil {
		t.Fatal(err)
	}
	if response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0); response.Code != 400 || !strings.Contains(response.Body.String(), "company_ineligible") {
		t.Fatal("non-Gold company accepted")
	}
	if err := connection.Model(&models.FairDateConfig{}).Where("1 = 1").Update("fair_days", "2020-11-17,2020-11-18").Error; err != nil {
		t.Fatal(err)
	}
	if response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", input, "test-service", 0); response.Code != 400 || !strings.Contains(response.Body.String(), "orders_closed") {
		t.Fatal("closed orders accepted")
	}
	var count int64
	if err := connection.Model(&models.LoungeOrder{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("invalid submission persisted an order")
	}
}

func TestLoungeInvalidResourceIDsAreRejected(t *testing.T) {
	for _, id := range []string{"0", "-1", "9007199254740992", "1 OR 1=1"} {
		for _, handler := range []http.HandlerFunc{GetLoungeCategory, GetLoungeOrder, DeleteLoungeCategory, RetryLoungeDelivery} {
			request := mux.SetURLVars(httptest.NewRequest("GET", "/", nil), map[string]string{"id": id})
			response := httptest.NewRecorder()
			handler(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("invalid ID %q was not rejected before database access", id)
			}
		}
	}
}

func TestLoungeRetryPreservesFailureAndUsesTestOrigin(t *testing.T) {
	connection, company, _, item := loungeTestDatabase(t)
	t.Setenv("K_SERVICE", "")
	t.Setenv("ORDER_INTERNAL_TEST_URL", "")
	t.Setenv("ORDER_INTERNAL_TEST_SECRET", "test-retry")
	response := loungeRequestTest(SubmitLoungeOrder, "POST", "/submit", loungeInput(company, item, "00000000-0000-4000-8000-000000000009"), "test-service", 0)
	var order models.LoungeOrder
	if response.Code != 201 || json.Unmarshal(response.Body.Bytes(), &order) != nil {
		t.Fatalf("submission failed: %s", response.Body)
	}
	if err := connection.Model(&order).Update("delivery_status", "failed").Error; err != nil {
		t.Fatal(err)
	}
	response = loungeRequestTest(RetryLoungeDelivery, "POST", "/retry", nil, "", order.ID)
	if response.Code != 503 {
		t.Fatalf("missing retry config was accepted: %s", response.Body)
	}
	if err := connection.First(&order, order.ID).Error; err != nil || order.DeliveryStatus != "failed" {
		t.Fatal("missing retry configuration cleared the recorded failure")
	}
	calls := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/order-delivery" || r.Header.Get("Authorization") != "Bearer test-retry" {
			http.Error(w, "Invalid retry request", 400)
			return
		}
		var input map[string]any
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			http.Error(w, "Invalid retry body", 400)
			return
		}
		calls <- input
		claim := loungeRequestTest(ClaimLoungeDelivery, "POST", "/claim", nil, "test-service", order.ID)
		var granted struct {
			Claimed bool   `json:"claimed"`
			Token   string `json:"claimToken"`
		}
		if claim.Code != 200 || json.Unmarshal(claim.Body.Bytes(), &granted) != nil || !granted.Claimed {
			http.Error(w, "Retry claim failed", 500)
			return
		}
		report := loungeRequestTest(ReportLoungeDelivery, "POST", "/delivery", map[string]any{"claimToken": granted.Token, "status": "sent"}, "test-service", order.ID)
		w.WriteHeader(report.Code)
	}))
	defer server.Close()
	t.Setenv("ORDER_INTERNAL_TEST_URL", server.URL)
	t.Setenv("ORDER_INTERNAL_URL", "https://must-not-call.invalid")
	response = loungeRequestTest(RetryLoungeDelivery, "POST", "/retry", nil, "", order.ID)
	if response.Code != 200 {
		t.Fatalf("test retry failed: %d %s", response.Code, response.Body)
	}
	input := <-calls
	if input["isTest"] != true || input["orderId"] != float64(order.ID) {
		t.Fatalf("wrong retry environment: %+v", input)
	}
	if err := connection.First(&order, order.ID).Error; err != nil || order.DeliveryStatus != "sent" {
		t.Fatal("retry did not record successful delivery")
	}
	if repeated := loungeRequestTest(RetryLoungeDelivery, "POST", "/retry", nil, "", order.ID); repeated.Code != 409 {
		t.Fatal("retry of delivered notification was accepted")
	}
}
