package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type eventroSourceWriteKey struct{}

func eventroSourceWriteRequest(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), eventroSourceWriteKey{}, true))
}

// A normal editor save acknowledges changed fields as CMS-owned, without
// altering the last imported source. Only a later source change can reopen review.
func acknowledgeEventroCMSEdits(tx *gorm.DB, r *http.Request, resource string, before, after any) error {
	if _, supported := eventroResourceSpecs[resource]; !supported {
		return nil
	}
	if sourceWrite, _ := r.Context().Value(eventroSourceWriteKey{}).(bool); sourceWrite {
		return nil
	}
	oldValues, newValues := eventroReviewValues(resource, before), eventroReviewValues(resource, after)
	changed := []string{}
	for field, value := range newValues {
		if !sameEventroValue(oldValues[field], value) {
			changed = append(changed, field)
		}
	}
	if len(changed) == 0 {
		return nil
	}
	id, err := strconv.ParseUint(getResourceID(after), 10, strconv.IntSize)
	if err != nil {
		return err
	}
	state, err := loadEventroSyncState(tx, resource, uint(id))
	if err != nil {
		return err
	}
	if len(state.Source) == 0 {
		return nil
	}
	stateJSON, _ := json.Marshal(state)
	var stateBefore models.EventroSyncState
	_ = json.Unmarshal(stateJSON, &stateBefore)
	if state.Overrides == nil {
		state.Overrides = map[string]bool{}
	}
	for _, field := range changed {
		state.Overrides[field] = true
		delete(state.Conflicts, field)
	}
	return saveEventroSyncStateWithAudit(tx, r, stateBefore, &state)
}

type eventroResourceSpec struct {
	newModel func() any
	fields   []string
	tags     []string
}

var eventroResourceSpecs = map[string]eventroResourceSpec{
	"exhibitors":         {func() any { return &models.Exhibitor{} }, []string{"name", "tier", "companyWebsite", "about", "logo", "cities", "industries", "employments", "programs"}, []string{"exhibitors", "industries", "employments", "programs"}},
	"events":             {func() any { return &models.Event{} }, []string{"name", "description", "eventStart", "eventEnd", "registrationEnd", "eventMaxCapacity", "location", "food", "openTo", "mapUrl", "signupLink", "registrationRequired"}, []string{"events"}},
	"profiles":           {func() any { return &models.Profile{} }, []string{"name", "rank", "title", "email", "photo"}, []string{"organization"}},
	"fairdates":          {func() any { return &models.FairDateConfig{} }, []string{"description", "fairDays", "irStart", "irEnd", "irAcceptance", "frStart", "frEnd", "eventsStart"}, []string{"dates"}},
	"recruitmentperiods": {func() any { return &models.RecruitmentPeriod{} }, []string{"name", "link", "startDate", "endDate"}, []string{"recruitment", "organization"}},
	"recruitmentroles":   {func() any { return &models.RecruitmentRole{} }, []string{"name", "description", "team_id", "recruitmentId"}, []string{"recruitment", "organization"}},
}

func eventroStructValues(entity any, fields []string) map[string]json.RawMessage {
	value := reflect.Indirect(reflect.ValueOf(entity))
	result := map[string]json.RawMessage{}
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		for _, allowed := range fields {
			if allowed == name {
				v := value.Field(i).Interface()
				switch date := v.(type) {
				case time.Time:
					v = date.UTC()
				case *time.Time:
					if date != nil {
						utc := date.UTC()
						v = &utc
					}
				}
				result[name], _ = json.Marshal(v)
			}
		}
	}
	return result
}

func mergeEventroValues(current, source map[string]json.RawMessage, state *models.EventroSyncState, isNew bool) map[string]json.RawMessage {
	if state.Source == nil {
		state.Source = map[string]json.RawMessage{}
	}
	if state.Overrides == nil {
		state.Overrides = map[string]bool{}
	}
	if state.Conflicts == nil {
		state.Conflicts = map[string]bool{}
	}
	// Archived/omitted fields are no longer current source values to offer for review.
	// Keep ownership so an archived CMS correction stays protected if the field returns.
	for field := range state.Source {
		if _, present := source[field]; !present {
			delete(state.Source, field)
			delete(state.Conflicts, field)
		}
	}
	result := map[string]json.RawMessage{}
	for field, value := range source {
		previous, known := state.Source[field]
		equal := sameEventroValue(current[field], value)
		manual := !isNew && (state.Overrides[field] || (known && !sameEventroValue(current[field], previous)) || (!known && !emptyEventroValue(current[field]) && !equal))
		if manual {
			state.Overrides[field] = true
			if equal {
				delete(state.Conflicts, field)
			} else if !known || !sameEventroValue(previous, value) {
				state.Conflicts[field] = true
			}
		} else {
			result[field] = value
			delete(state.Conflicts, field)
		}
		state.Source[field] = value
	}
	return result
}

func applyEventroStructValue(entity any, field string, value json.RawMessage) error {
	object, err := json.Marshal(map[string]json.RawMessage{field: value})
	if err != nil {
		return err
	}
	return json.Unmarshal(object, entity)
}

func loadEventroSyncState(tx *gorm.DB, resource string, id uint) (models.EventroSyncState, error) {
	state := models.EventroSyncState{Resource: resource, RecordID: id, FairID: "", UnmappedProducts: []models.EventroOrderedProduct{}}
	err := tx.Where("resource = ? AND record_id = ?", resource, id).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return state, err
}

func saveEventroSyncStateWithAudit(tx *gorm.DB, r *http.Request, before models.EventroSyncState, after *models.EventroSyncState) error {
	old, _ := json.Marshal(before)
	newValue, _ := json.Marshal(after)
	if sameEventroValue(old, newValue) {
		return nil
	}
	if err := tx.Save(after).Error; err != nil {
		return err
	}
	return audit.LogUpdate(tx, r, "eventro_sync_states", fmt.Sprintf("%s:%d", after.Resource, after.RecordID), json.RawMessage(old), after)
}

// syncEventroRecordWithAudit runs inside the caller's transaction and updates only
// the allowlisted import fields. Source state and audited content commit together.
func syncEventroRecordWithAudit(tx *gorm.DB, r *http.Request, resource string, incoming any, identityColumn string, identity any, fallback map[string]any, fields []string) (uint, bool, bool, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("eventro:%s:%v", resource, identity)).Error; err != nil {
		return 0, false, false, err
	}
	spec := eventroResourceSpecs[resource]
	existing := spec.newModel()
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(identityColumn+" = ?", identity).First(existing).Error
	if err == gorm.ErrRecordNotFound && len(fallback) > 0 {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(identityColumn + " IS NULL")
		for column, value := range fallback {
			query = query.Where(column+" = ?", value)
		}
		err = query.First(existing).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return 0, false, false, err
	}
	isNew := err == gorm.ErrRecordNotFound
	if fields == nil {
		fields = spec.fields
	}
	if isNew {
		if err := tx.Omit(clause.Associations).Create(incoming).Error; err != nil {
			return 0, false, false, err
		}
		id := uint(reflect.Indirect(reflect.ValueOf(incoming)).FieldByName("ID").Uint())
		state := models.EventroSyncState{Resource: resource, RecordID: id, FairID: r.URL.Query().Get("fairId"), Source: eventroStructValues(incoming, fields), Overrides: map[string]bool{}, Conflicts: map[string]bool{}, UnmappedProducts: []models.EventroOrderedProduct{}}
		if err := saveEventroSyncStateWithAudit(tx, r, models.EventroSyncState{}, &state); err != nil {
			return 0, false, false, err
		}
		return id, true, false, audit.LogCreate(tx, r, resource, id, incoming)
	}
	id := uint(reflect.Indirect(reflect.ValueOf(existing)).FieldByName("ID").Uint())
	state, err := loadEventroSyncState(tx, resource, id)
	if err != nil {
		return 0, false, false, err
	}
	stateJSON, _ := json.Marshal(state)
	var stateBefore models.EventroSyncState
	_ = json.Unmarshal(stateJSON, &stateBefore)
	updates := mergeEventroValues(eventroStructValues(existing, spec.fields), eventroStructValues(incoming, fields), &state, false)
	beforeJSON, _ := json.Marshal(existing)
	identityJSON, _ := json.Marshal(identity)
	identityName := map[string]string{"eventro_id": "eventroId", "eventro_role_id": "eventroRoleId", "eventro_key": "eventro_key"}[identityColumn]
	if err := applyEventroStructValue(existing, identityName, identityJSON); err != nil {
		return 0, false, false, err
	}
	for field, value := range updates {
		if err := applyEventroStructValue(existing, field, value); err != nil {
			return 0, false, false, err
		}
	}
	afterJSON, _ := json.Marshal(existing)
	changed := string(beforeJSON) != string(afterJSON)
	// Stable external identity is metadata, not an editable imported field.
	identityField := identityColumn
	columns := []string{identityField}
	values := map[string]any{identityField: identity}
	statement := &gorm.Statement{DB: tx}
	if err := statement.Parse(existing); err != nil {
		return 0, false, false, err
	}
	for _, f := range statement.Schema.Fields {
		name := strings.Split(f.StructField.Tag.Get("json"), ",")[0]
		if _, ok := updates[name]; ok {
			v, _ := f.ValueOf(r.Context(), reflect.ValueOf(existing))
			values[f.DBName] = v
			columns = append(columns, f.DBName)
		}
	}
	if err := tx.Model(existing).Select(columns).Updates(values).Error; err != nil {
		return 0, false, false, err
	}
	if err := saveEventroSyncStateWithAudit(tx, r, stateBefore, &state); err != nil {
		return 0, false, false, err
	}
	if changed {
		if err := audit.LogUpdate(tx, r, resource, id, json.RawMessage(beforeJSON), existing); err != nil {
			return 0, false, false, err
		}
	}
	return id, false, changed, nil
}

// Reject invalid dates instead of importing the current time as a replacement.
func validEventroTimestamp(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}
