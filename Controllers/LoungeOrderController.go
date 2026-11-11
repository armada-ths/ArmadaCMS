package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

type loungeSubmission struct {
	IdempotencyKey string `json:"idempotencyKey"`
	ExhibitorID    uint   `json:"exhibitorId"`
	Items          []struct {
		ID       string `json:"id"`
		Quantity int64  `json:"quantity"`
	} `json:"items"`
}

type loungeOrderStatus struct {
	Status string `json:"status"`
}

type loungeOrderError string

func (e loungeOrderError) Error() string { return string(e) }

var loungeUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func loungeService(w http.ResponseWriter, r *http.Request) (bool, bool) {
	live, test := os.Getenv("LOUNGE_ORDER_SUBMISSION_SECRET"), os.Getenv("LOUNGE_ORDER_TEST_SUBMISSION_SECRET")
	w.Header().Set("Cache-Control", "no-store")
	if live != "" && live == test {
		log.Print("lounge live/test submission credentials must differ")
		writeJSONResponse(w, 503, map[string]string{"errorCode": "submission_failed"})
		return false, false
	}
	provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		writeJSONResponse(w, 401, map[string]string{"errorCode": "access_required"})
		return false, false
	}
	if live != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(live)) == 1 {
		return false, true
	}
	if test != "" && test != live && subtle.ConstantTimeCompare([]byte(provided), []byte(test)) == 1 {
		return true, true
	}
	writeJSONResponse(w, 401, map[string]string{"errorCode": "access_required"})
	return false, false
}

func loungeOpen(parts []string, hours [2]models.FairOpeningHours, now time.Time, closeMinutes int) bool {
	zone, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return false
	}
	if len(parts) < 2 {
		return false
	}
	parse := func(value string) (time.Time, error) {
		value = strings.TrimSpace(value)
		if t, err := time.ParseInLocation("2006-01-02", value, zone); err == nil {
			return t, nil
		}
		if t, err := time.Parse(time.RFC3339, value); err == nil {
			t = t.In(zone)
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, zone), nil
		}
		return time.Time{}, errors.New("invalid fair date")
	}
	first, err1 := parse(parts[0])
	second, err2 := parse(parts[1])
	if err1 != nil || err2 != nil || !second.After(first) {
		return false
	}
	now = now.In(zone)
	lead := time.Duration(closeMinutes) * time.Minute
	for i, day := range []time.Time{first, second} {
		if now.Format("2006-01-02") == day.Format("2006-01-02") {
			date := day.Format("2006-01-02")
			open, openErr := time.ParseInLocation("2006-01-02 15:04", date+" "+hours[i].OpensAt, zone)
			close, closeErr := time.ParseInLocation("2006-01-02 15:04", date+" "+hours[i].ClosesAt, zone)
			cutoff := close.Add(-lead)
			return openErr == nil && closeErr == nil && cutoff.After(open) && !now.Before(open) && !now.After(cutoff)
		}
	}
	return false
}

// GetLoungeOrderAvailability returns the current CMS ordering window.
// @Summary Check whether lounge ordering is open
// @Tags lounge-orders
// @Success 200 {object} map[string]bool
// @Router /lounge-orders/availability [get]
func GetLoungeOrderAvailability(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var dates models.FairDateConfig
	if err := db.DB.Order("id DESC").First(&dates).Error; err != nil {
		writeJSONResponse(w, 503, map[string]string{"errorCode": "availability_unavailable"})
		return
	}
	writeJSONResponse(w, 200, map[string]bool{"open": loungeOpen(dates.ToFairDate().Fair.Days, dates.OpeningHours(), time.Now(), dates.LoungeClosingLeadMinutes())})
}

// SubmitLoungeOrder persists an authenticated, idempotent order before delivery.
// @Summary Submit lounge order
// @Tags lounge-orders
// @Accept json
// @Produce json
// @Param body body loungeSubmission true "Order and idempotency key"
// @Success 200 {object} models.LoungeOrder
// @Success 201 {object} models.LoungeOrder
// @Security BearerAuth
// @Router /lounge-orders/submit [post]
func SubmitLoungeOrder(w http.ResponseWriter, r *http.Request) {
	submitLoungeOrder(w, r, time.Now)
}

func submitLoungeOrder(w http.ResponseWriter, r *http.Request, now func() time.Time) {
	test, ok := loungeService(w, r)
	if !ok {
		return
	}
	var input loungeSubmission
	if !decodeLoungeJSON(w, r, &input) {
		return
	}
	if !loungeUUID.MatchString(input.IdempotencyKey) || input.ExhibitorID == 0 || input.ExhibitorID > 9007199254740991 || len(input.Items) == 0 || len(input.Items) > 1000 {
		writeJSONResponse(w, 400, map[string]string{"errorCode": "invalid_order"})
		return
	}
	payload, _ := json.Marshal(input)
	hash := sha256.Sum256(payload)
	fingerprint := hex.EncodeToString(hash[:])
	var order models.LoungeOrder
	var existing bool
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Serialize identical keys even when no order row exists yet.
		lockKey := fmt.Sprintf("%t:%s", test, input.IdempotencyKey)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", lockKey).Error; err != nil {
			return err
		}
		err := tx.Where("is_test = ? AND idempotency_key = ?", test, input.IdempotencyKey).Preload("Lines", func(tx *gorm.DB) *gorm.DB { return tx.Order("position") }).First(&order).Error
		if err == nil {
			if order.Fingerprint != fingerprint {
				return loungeOrderError("invalid_order")
			}
			existing = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Exec("SELECT id FROM lounge_catalogue_state WHERE id = 1 FOR SHARE").Error; err != nil {
			return err
		}
		var dates models.FairDateConfig
		if err := tx.Order("id DESC").First(&dates).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return loungeOrderError("availability_unavailable")
			}
			return err
		}
		if !loungeOpen(dates.ToFairDate().Fair.Days, dates.OpeningHours(), now(), dates.LoungeClosingLeadMinutes()) {
			return loungeOrderError("orders_closed")
		}
		var company models.Exhibitor
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ? AND tier = ?", input.ExhibitorID, models.TierGold).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return loungeOrderError("company_ineligible")
			}
			return err
		}
		order = models.LoungeOrder{CompanyID: company.ID, CompanyName: company.Name, IsTest: test, Status: "pending", DeliveryStatus: "pending", IdempotencyKey: input.IdempotencyKey, Fingerprint: fingerprint, Lines: []models.LoungeOrderLine{}}
		itemIDs := make([]uint, 0, len(input.Items))
		seen := make(map[uint]bool, len(input.Items))
		for _, line := range input.Items {
			id, err := strconv.ParseUint(line.ID, 10, strconv.IntSize)
			if err != nil || id == 0 || id > 9007199254740991 || line.Quantity <= 0 || line.Quantity > 9007199254740991 || seen[uint(id)] {
				return loungeOrderError("invalid_order")
			}
			seen[uint(id)] = true
			itemIDs = append(itemIDs, uint(id))
		}
		var items []models.LoungeItem
		if err := tx.Where("id IN ?", itemIDs).Find(&items).Error; err != nil {
			return err
		}
		if len(items) != len(itemIDs) {
			return loungeOrderError("items_changed")
		}
		itemsByID := make(map[uint]models.LoungeItem, len(items))
		categoryIDs := make([]uint, 0, len(items))
		for _, item := range items {
			itemsByID[item.ID] = item
			categoryIDs = append(categoryIDs, item.CategoryID)
		}
		var categories []models.LoungeCategory
		if err := tx.Where("id IN ?", categoryIDs).Find(&categories).Error; err != nil {
			return err
		}
		categoriesByID := make(map[uint]models.LoungeCategory, len(categories))
		for _, category := range categories {
			categoriesByID[category.ID] = category
		}
		for position, line := range input.Items {
			item := itemsByID[itemIDs[position]]
			category, found := categoriesByID[item.CategoryID]
			if !found || item.SoldOut || (item.MaxPerOrder != nil && line.Quantity > *item.MaxPerOrder) {
				return loungeOrderError("items_changed")
			}
			order.Lines = append(order.Lines, models.LoungeOrderLine{ItemID: item.ID, CategoryID: category.ID, ItemName: item.Name, ItemNameSv: item.NameSv, CategoryName: category.Name, CategoryNameSv: category.NameSv, Quantity: line.Quantity, Position: position})
		}
		return createWithAuditTx(tx, r, &order)
	})
	if err != nil {
		var code loungeOrderError
		if errors.As(err, &code) {
			writeJSONResponse(w, 400, map[string]string{"errorCode": string(code)})
			return
		}
		log.Printf("lounge order persistence failed: %v", err)
		writeJSONResponse(w, 503, map[string]string{"errorCode": "submission_failed"})
		return
	}
	if existing {
		writeJSONResponse(w, 200, order)
	} else {
		writeCreatedJSONResponse(w, order)
	}
}

func createWithAuditTx(tx *gorm.DB, r *http.Request, order *models.LoungeOrder) error {
	return createWithAuditInTransaction(tx, r, "lounge-orders", order, func(tx *gorm.DB) error { return tx.Create(order).Error }, nil)
}

func updateLoungeOrderWithAudit(tx *gorm.DB, r *http.Request, before models.LoungeOrder, after *models.LoungeOrder, updates map[string]any) error {
	return updateWithAuditInTransaction(tx, r, "lounge-orders", after.ID, before, after, func(tx *gorm.DB) error {
		return tx.Model(after).Updates(updates).Error
	}, nil)
}

func loungeOrderQuery(r *http.Request) (*gorm.DB, map[string]any, error) {
	query := db.DB.Model(&models.LoungeOrder{})
	filter := map[string]any{}
	if raw := r.URL.Query().Get("filter"); raw != "" {
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.UseNumber()
		if !json.Valid([]byte(raw)) {
			return nil, nil, errors.New("invalid filter")
		}
		if err := decoder.Decode(&filter); err != nil {
			return nil, nil, errors.New("invalid filter")
		}
	}
	for key := range filter {
		if key != "companyId" && key != "categoryId" && key != "itemId" && key != "status" && key != "isTest" {
			return nil, nil, errors.New("unsupported filter")
		}
	}
	for _, key := range []string{"companyId", "categoryId", "itemId"} {
		if value, exists := filter[key]; exists {
			id, err := strconv.ParseUint(fmt.Sprint(value), 10, 64)
			if err != nil || id == 0 || id > 9007199254740991 {
				return nil, nil, errors.New("invalid ID filter")
			}
			filter[key] = id
		}
	}
	test := false
	if value, exists := filter["isTest"]; exists {
		parsed, ok := value.(bool)
		if !ok {
			return nil, nil, errors.New("invalid test filter")
		}
		test = parsed
	}
	query = query.Where("is_test = ?", test)
	if status, exists := filter["status"]; exists {
		if status != "pending" && status != "completed" {
			return nil, nil, errors.New("invalid status")
		}
		query = query.Where("status = ?", status)
	}
	if company, exists := filter["companyId"]; exists {
		query = query.Where("company_id = ?", company)
	}
	conditions := []string{"lounge_order_lines.order_id = lounge_orders.id"}
	arguments := []any{}
	for key, column := range map[string]string{"categoryId": "category_id", "itemId": "item_id"} {
		if value, exists := filter[key]; exists {
			conditions = append(conditions, "lounge_order_lines."+column+" = ?")
			arguments = append(arguments, value)
		}
	}
	if len(arguments) > 0 {
		query = query.Where("EXISTS (SELECT 1 FROM lounge_order_lines WHERE "+strings.Join(conditions, " AND ")+")", arguments...)
	}
	return query, filter, nil
}

// ListLoungeOrders lists private orders with historical filters.
// @Summary List lounge orders
// @Tags lounge-orders
// @Param filter query string false "Company, category, item, status, isTest"
// @Success 200 {array} models.LoungeOrder
// @Security BearerAuth
// @Router /lounge-orders [get]
func ListLoungeOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	query, _, err := loungeOrderQuery(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		loungeError(w, err)
		return
	}
	params, _ := utils.ParseListParams(r.URL.Query())
	column, direction := "id", "DESC"
	if len(params.Sort) == 2 {
		columns := map[string]string{"id": "id", "createdAt": "created_at", "companyName": "company_name", "status": "status", "deliveryStatus": "delivery_status"}
		if value, ok := columns[params.Sort[0]]; ok {
			column = value
		}
		if params.Sort[1] == "ASC" {
			direction = "ASC"
		}
	}
	items := []models.LoungeOrder{}
	start, end := params.Range[0], params.Range[1]
	if start < 0 || end < start || end-start >= 1000 {
		http.Error(w, "Invalid range", 400)
		return
	}
	if column == "delivery_status" {
		column = "CASE WHEN delivery_status = 'sending' AND claim_expires_at <= CURRENT_TIMESTAMP THEN 'unknown' ELSE delivery_status END"
	}
	if err := query.Order(column+" "+direction).Order("id DESC").Offset(start).Limit(end-start+1).Preload("Lines", func(tx *gorm.DB) *gorm.DB { return tx.Order("position") }).Find(&items).Error; err != nil {
		loungeError(w, err)
		return
	}
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("lounge-orders %d-%d/%d", start, start+len(items)-1, total))
	now := time.Now()
	for i := range items {
		items[i].DeliveryStatus = loungeDeliveryStatus(items[i], now)
	}
	writeJSONResponse(w, 200, items)
}

// GetLoungeOrder returns an immutable order snapshot.
// @Summary Get lounge order
// @Tags lounge-orders
// @Param id path int true "Order ID"
// @Success 200 {object} models.LoungeOrder
// @Security BearerAuth
// @Router /lounge-orders/{id} [get]
func GetLoungeOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, ok := loungePathID(w, r)
	if !ok {
		return
	}
	var order models.LoungeOrder
	if err := db.DB.Preload("Lines", func(tx *gorm.DB) *gorm.DB { return tx.Order("position") }).First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Order not found", 404)
		} else {
			loungeError(w, err)
		}
		return
	}
	order.DeliveryStatus = loungeDeliveryStatus(order, time.Now())
	writeJSONResponse(w, 200, order)
}

func loungeDeliveryStatus(order models.LoungeOrder, now time.Time) string {
	if order.DeliveryStatus == "sending" && order.ClaimExpiresAt != nil && !order.ClaimExpiresAt.After(now) {
		return "unknown"
	}
	return order.DeliveryStatus
}

// UpdateLoungeOrder changes operational status, never order lines.
// @Summary Change lounge order status
// @Tags lounge-orders
// @Accept json
// @Param id path int true "Order ID"
// @Param body body loungeOrderStatus true "Operational status only"
// @Success 200 {object} models.LoungeOrder
// @Security BearerAuth
// @Router /lounge-orders/{id} [put]
func UpdateLoungeOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, ok := loungePathID(w, r)
	if !ok {
		return
	}
	var input loungeOrderStatus
	if !decodeLoungeJSON(w, r, &input) {
		return
	}
	if input.Status != "pending" && input.Status != "completed" {
		http.Error(w, "Invalid order status", 400)
		return
	}
	var order models.LoungeOrder
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Lines").First(&order, id).Error; err != nil {
			return err
		}
		before := order
		order.Status = input.Status
		return updateLoungeOrderWithAudit(tx, r, before, &order, map[string]any{"status": input.Status})
	})
	if err != nil {
		loungeError(w, err)
		return
	}
	order.DeliveryStatus = loungeDeliveryStatus(order, time.Now())
	writeJSONResponse(w, 200, order)
}

// LoungeOrderTotals counts matching lines across all filtered orders.
// @Summary Lounge quantity totals
// @Tags lounge-orders
// @Success 200 {array} map[string]interface{}
// @Security BearerAuth
// @Router /lounge-orders/totals [get]
func LoungeOrderTotals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	query, filter, err := loungeOrderQuery(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	lines := db.DB.Table("lounge_order_lines").Where("order_id IN (?)", query.Select("id"))
	if value, ok := filter["categoryId"]; ok {
		lines = lines.Where("category_id = ?", value)
	}
	if value, ok := filter["itemId"]; ok {
		lines = lines.Where("item_id = ?", value)
	}
	type total struct {
		ItemID       uint   `json:"itemId"`
		CategoryID   uint   `json:"categoryId"`
		ItemName     string `json:"itemName"`
		CategoryName string `json:"categoryName"`
		Quantity     string `json:"quantity"`
	}
	totals := []total{}
	if err := lines.Select("item_id, category_id, (array_agg(item_name ORDER BY order_id DESC))[1] AS item_name, (array_agg(category_name ORDER BY order_id DESC))[1] AS category_name, SUM(quantity)::text AS quantity").Group("item_id, category_id").Order("category_name, item_name").Scan(&totals).Error; err != nil {
		loungeError(w, err)
		return
	}
	writeJSONResponse(w, 200, totals)
}

// LoungeOrderFilterOptions includes historic names and deleted catalogue entries.
// @Summary Lounge historical filter options
// @Tags lounge-orders
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /lounge-orders/filter-options [get]
func LoungeOrderFilterOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	type option struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	companies, categories, items := []option{}, []option{}, []option{}
	for _, spec := range []struct {
		table, id, name string
		target          *[]option
	}{
		{"lounge_orders", "company_id", "company_name", &companies},
		{"lounge_order_lines", "category_id", "category_name", &categories},
		{"lounge_order_lines", "item_id", "item_name", &items},
	} {
		if err := db.DB.Table(spec.table).Select(spec.id + " AS id, MAX(" + spec.name + ") AS name").Group(spec.id).Order("name").Scan(spec.target).Error; err != nil {
			loungeError(w, err)
			return
		}
	}
	writeJSONResponse(w, 200, map[string]any{"companies": companies, "categories": categories, "items": items})
}
