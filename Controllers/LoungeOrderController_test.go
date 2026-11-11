package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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

func submitLoungeOrderDuringFair(w http.ResponseWriter, r *http.Request) {
	submitLoungeOrder(w, r, func() time.Time {
		return time.Date(2099, time.November, 17, 11, 0, 0, 0, time.UTC)
	})
}

func loungeInput(company, item uint, key string) map[string]any {
	return map[string]any{"idempotencyKey": key, "exhibitorId": company, "items": []map[string]any{{"id": fmt.Sprint(item), "quantity": 2}}}
}

func TestLoungeOrderIDFilters(t *testing.T) {
	connection, err := gorm.Open(postgres.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	old := db.DB
	db.DB = connection
	t.Cleanup(func() {
		db.DB = old
		sql, sqlErr := connection.DB()
		if sqlErr == nil {
			_ = sql.Close()
		}
	})
	for _, key := range []string{"companyId", "categoryId", "itemId"} {
		for _, value := range []string{"900000001", "9007199254740991", `"900000001"`} {
			t.Run(key+"/"+value, func(t *testing.T) {
				raw := fmt.Sprintf(`{"%s":%s}`, key, value)
				request := httptest.NewRequest("GET", "/orders?filter="+url.QueryEscape(raw), nil)
				query, filters, queryErr := loungeOrderQuery(request)
				if queryErr != nil {
					t.Fatal(queryErr)
				}
				expected := uint64(900000001)
				if value == "9007199254740991" {
					expected = 9007199254740991
				}
				if filters[key] != expected {
					t.Fatalf("ID lost precision: %v", filters[key])
				}
				statement := query.Find(&[]models.LoungeOrder{}).Statement
				if !strings.Contains(fmt.Sprint(statement.Vars), fmt.Sprint(expected)) {
					t.Fatalf("ID missing from query: %v", statement.Vars)
				}
			})
		}
	}
	for _, raw := range []string{
		`{"companyId":9007199254740992}`, `{"companyId":1.5}`, `{"companyId":0}`,
		`{"companyId":true}`, `{"companyId":null}`, `{"companyId":-1}`,
		`{"companyId":1} {}`, `{"companyId":`,
	} {
		request := httptest.NewRequest("GET", "/orders?filter="+url.QueryEscape(raw), nil)
		if _, _, err := loungeOrderQuery(request); err == nil {
			t.Fatalf("accepted invalid filter: %s", raw)
		}
	}
}

var loungeTestHours = [2]models.FairOpeningHours{
	{OpensAt: "10:00", ClosesAt: "16:00"},
	{OpensAt: "10:00", ClosesAt: "15:00"},
}

func TestLoungeOpeningBoundaries(t *testing.T) {
	zone, _ := time.LoadLocation("Europe/Stockholm")
	for _, example := range []struct {
		time string
		open bool
	}{
		{"2026-11-16T12:00:00", false}, {"2026-11-17T09:59:59", false}, {"2026-11-17T10:00:00", true}, {"2026-11-17T15:30:00", true}, {"2026-11-17T15:30:01", false},
		{"2026-11-18T09:59:59", false}, {"2026-11-18T10:00:00", true}, {"2026-11-18T14:30:00", true},
		{"2026-11-18T14:30:01", false}, {"2026-11-19T10:00:00", false},
	} {
		now, _ := time.ParseInLocation("2006-01-02T15:04:05", example.time, zone)
		if loungeOpen([]string{"2026-11-17", "2026-11-18"}, loungeTestHours, now, models.DefaultLoungeOrderCloseMinutes) != example.open {
			t.Fatalf("incorrect opening at %s", example.time)
		}
	}
	if loungeOpen([]string{"invalid", "2026-11-18"}, loungeTestHours, time.Now(), models.DefaultLoungeOrderCloseMinutes) || loungeOpen([]string{"2026-11-18", "2026-11-17"}, loungeTestHours, time.Now(), models.DefaultLoungeOrderCloseMinutes) {
		t.Fatal("invalid dates accepted")
	}
}

func TestLoungeConfigurableClosingLead(t *testing.T) {
	days := []string{"2026-11-17", "2026-11-18"}
	zone, _ := time.LoadLocation("Europe/Stockholm")
	for _, example := range []struct {
		day, hour, minute, lead int
		open                    bool
	}{
		{17, 15, 1, 60, false},
		{18, 14, 0, 60, true},
		{18, 14, 1, 60, false},
		{18, 15, 0, 0, true},
		{18, 15, 1, 0, false},
		{18, 10, 0, 300, false},
	} {
		now := time.Date(2026, time.November, example.day, example.hour, example.minute, 0, 0, zone)
		if loungeOpen(days, loungeTestHours, now, example.lead) != example.open {
			t.Errorf("unexpected availability at %s with %d minutes lead", now, example.lead)
		}
	}
}

func TestLoungeUsesConfiguredFairHours(t *testing.T) {
	days := []string{"2026-11-17", "2026-11-18"}
	hours := [2]models.FairOpeningHours{{OpensAt: "11:15", ClosesAt: "17:00"}, {OpensAt: "09:30", ClosesAt: "14:15"}}
	zone, _ := time.LoadLocation("Europe/Stockholm")
	for _, example := range []struct {
		day, hour, minute int
		open              bool
	}{
		{17, 11, 14, false}, {17, 11, 15, true}, {17, 16, 30, true}, {17, 16, 31, false},
		{18, 9, 29, false}, {18, 9, 30, true}, {18, 13, 45, true}, {18, 13, 46, false},
	} {
		now := time.Date(2026, time.November, example.day, example.hour, example.minute, 0, 0, zone)
		if loungeOpen(days, hours, now, 30) != example.open {
			t.Errorf("unexpected availability at %s", now)
		}
	}
}

func TestLoungePersistenceIdempotencyAndHistory(t *testing.T) {
	connection, company, category, item := loungeTestDatabase(t)
	input := loungeInput(company, item, "00000000-0000-4000-8000-000000000001")
	response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/lounge-orders/submit", input, "test-service", 0)
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
	again := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/lounge-orders/submit", input, "test-service", 0)
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
	if changed := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/lounge-orders/submit", input, "test-service", 0); changed.Code != 400 {
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
			results <- loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/lounge-orders/submit", input, "test-service", 0)
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
	var calls atomic.Int32
	t.Setenv("SLACK_ORDER_TEST_HOOK_URL", "https://hooks.slack.com/services/fixture/test/key")
	client := &http.Client{Transport: loungeTestTransport(func(_ *http.Request) (*http.Response, error) {
		calls.Add(1)
		return loungeSlackResponse(http.StatusBadRequest), nil
	})}
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			errs <- dispatchLoungeOrders(httptest.NewRequest(http.MethodPost, "/dispatch", nil), client)
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("sent %d notifications for one order", calls.Load())
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
	if unauthorized := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "", 0); unauthorized.Code != 401 {
		t.Fatal("unauthenticated submission accepted")
	}
	if err := connection.Model(&models.LoungeItem{}).Where("id = ?", item).Update("sold_out", true).Error; err != nil {
		t.Fatal(err)
	}
	if failed := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0); failed.Code != 400 || !strings.Contains(failed.Body.String(), "items_changed") {
		t.Fatal("sold-out item accepted")
	}
	connection.Model(&models.LoungeItem{}).Where("id = ?", item).Update("sold_out", false)
	response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "live-service", 0)
	if response.Code != 201 {
		t.Fatalf("live submit: %s", response.Body)
	}
	var order models.LoungeOrder
	_ = json.Unmarshal(response.Body.Bytes(), &order)
	input["idempotencyKey"] = "00000000-0000-4000-8000-000000000004"
	loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0)
	totals := loungeRequestTest(LoungeOrderTotals, "GET", "/totals", nil, "", 0)
	if totals.Code != 200 || !strings.Contains(totals.Body.String(), `"quantity":"2"`) {
		t.Fatalf("test/live totals incorrect: %s", totals.Body)
	}
	completed := loungeRequestTest(UpdateLoungeOrder, "PUT", "/order", map[string]string{"status": "completed"}, "", order.ID)
	if completed.Code != 200 {
		t.Fatalf("status update: %s", completed.Body)
	}
	totals = loungeRequestTest(LoungeOrderTotals, "GET", "/totals?filter="+url.QueryEscape(`{"status":"completed"}`), nil, "", 0)
	if totals.Code != 200 || !strings.Contains(totals.Body.String(), `"quantity":"2"`) {
		t.Fatalf("completed order missing from totals: %s", totals.Body)
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
	response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", loungeInput(company, item, "00000000-0000-4000-8000-000000000005"), "test-service", 0)
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
		response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "live-service", 0)
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
		response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0)
		if response.Code != 400 {
			t.Fatalf("quantity %v accepted: %d %s", quantity, response.Code, response.Body)
		}
	}
	input := loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
	input["items"] = []map[string]any{{"id": fmt.Sprint(item), "quantity": 1}, {"id": fmt.Sprint(item), "quantity": 1}}
	if response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0); response.Code != 400 {
		t.Fatal("duplicate lines accepted")
	}
	input = loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
	input["items"] = []map[string]any{{"id": "9007199254740991", "quantity": 1}}
	if response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0); response.Code != 400 || !strings.Contains(response.Body.String(), "items_changed") {
		t.Fatal("missing item accepted")
	}
	input = loungeInput(company, item, "00000000-0000-4000-8000-000000000008")
	if err := connection.Model(&models.Exhibitor{}).Where("id = ?", company).Update("tier", "Silver").Error; err != nil {
		t.Fatal(err)
	}
	if response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0); response.Code != 400 || !strings.Contains(response.Body.String(), "company_ineligible") {
		t.Fatal("non-Gold company accepted")
	}
	if err := connection.Model(&models.FairDateConfig{}).Where("1 = 1").Update("fair_days", "2020-11-17,2020-11-18").Error; err != nil {
		t.Fatal(err)
	}
	if response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", input, "test-service", 0); response.Code != 400 || !strings.Contains(response.Body.String(), "orders_closed") {
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

type loungeTestTransport func(*http.Request) (*http.Response, error)

func (transport loungeTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return transport(r)
}

func loungeSlackResponse(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}
}

func TestLoungeRetryQueuesWithoutClearingUnknownDelivery(t *testing.T) {
	connection, company, _, item := loungeTestDatabase(t)
	testHook := "https://hooks.slack.com/services/fixture/test/key"
	t.Setenv("SLACK_ORDER_TEST_HOOK_URL", testHook)
	t.Setenv("SLACK_ORDER_HOOK_URL", "https://hooks.slack.com/services/fixture/live/key")
	response := loungeRequestTest(submitLoungeOrderDuringFair, "POST", "/submit", loungeInput(company, item, "00000000-0000-4000-8000-000000000009"), "test-service", 0)
	var order models.LoungeOrder
	if response.Code != 201 || json.Unmarshal(response.Body.Bytes(), &order) != nil {
		t.Fatalf("submission failed: %s", response.Body)
	}
	if err := connection.Model(&order).Updates(map[string]any{
		"delivery_status": "sending", "claim_expires_at": time.Now().Add(-time.Minute),
	}).Error; err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client := &http.Client{Transport: loungeTestTransport(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		var payload struct{ Text string }
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			return nil, err
		}
		if request.URL.String() != testHook || !strings.Contains(payload.Text, "[STAGING TEST]") ||
			!strings.Contains(payload.Text, "Gold AB") || !strings.Contains(payload.Text, "- Coffee x 2") {
			return nil, fmt.Errorf("incorrect notification destination or text: %s", payload.Text)
		}
		return loungeSlackResponse(http.StatusOK), nil
	})}
	run := func() {
		t.Helper()
		if err := dispatchLoungeOrders(httptest.NewRequest(http.MethodPost, "/dispatch", nil), client); err != nil {
			t.Fatal(err)
		}
	}
	run()
	if calls.Load() != 0 {
		t.Fatal("expired claim was retried automatically")
	}
	for _, response := range []*httptest.ResponseRecorder{
		loungeRequestTest(ListLoungeOrders, "GET", "/orders?filter="+url.QueryEscape(`{"isTest":true}`), nil, "", 0),
		loungeRequestTest(GetLoungeOrder, "GET", "/order", nil, "", order.ID),
	} {
		if response.Code != 200 || !strings.Contains(response.Body.String(), `"deliveryStatus":"unknown"`) {
			t.Fatalf("expired delivery displayed incorrectly: %d %s", response.Code, response.Body)
		}
	}
	response = loungeRequestTest(RetryLoungeDelivery, "POST", "/retry", nil, "", order.ID)
	if response.Code != http.StatusAccepted {
		t.Fatalf("retry was not queued: %d %s", response.Code, response.Body)
	}
	if err := connection.First(&order, order.ID).Error; err != nil ||
		!order.DeliveryRetryRequested || loungeDeliveryStatus(order, time.Now()) != "unknown" {
		t.Fatal("queued retry cleared the earlier delivery state")
	}
	run()
	if calls.Load() != 1 {
		t.Fatal("queued retry was not dispatched")
	}
	if err := connection.First(&order, order.ID).Error; err != nil ||
		order.DeliveryStatus != "sent" || order.DeliveryRetryRequested {
		t.Fatal("worker did not record the successful retry")
	}
	if repeated := loungeRequestTest(RetryLoungeDelivery, "POST", "/retry", nil, "", order.ID); repeated.Code != http.StatusConflict {
		t.Fatal("retry of delivered notification was accepted")
	}
}

func TestLoungeSlackEnvironmentIsolation(t *testing.T) {
	live := "https://hooks.slack.com/services/fixture/live/key"
	t.Setenv("SLACK_ORDER_HOOK_URL", live)
	client := &http.Client{Transport: loungeTestTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected outbound Slack request to %s", request.URL.Host)
		return nil, nil
	})}
	for _, webhook := range []string{"", live, "https://example.com/services/test"} {
		t.Setenv("SLACK_ORDER_TEST_HOOK_URL", webhook)
		status, reason := sendLoungeSlack(t.Context(), client, models.LoungeOrder{IsTest: true})
		if status != "failed" || reason != "not_configured" {
			t.Fatalf("unexpected outcome: %s %s", status, reason)
		}
	}
	t.Setenv("SLACK_ORDER_TEST_HOOK_URL", "https://hooks.slack.com/services/fixture/test/key")
	client.Transport = loungeTestTransport(func(_ *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("simulated timeout")
	})
	status, reason := sendLoungeSlack(t.Context(), client, models.LoungeOrder{IsTest: true})
	if status != "unknown" || reason != "slack_unreachable" {
		t.Fatalf("timeout was not marked unknown: %s %s", status, reason)
	}
}
