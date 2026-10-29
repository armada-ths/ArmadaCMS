package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func persistEventroExhibitorCategories(tx *gorm.DB, r *http.Request, ex *models.Exhibitor, before models.Exhibitor) error {
	for i := range ex.Industries {
		result := tx.Where("name = ?", ex.Industries[i].Name).FirstOrCreate(&ex.Industries[i])
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := audit.LogCreate(tx, r, "industries", ex.Industries[i].ID, ex.Industries[i]); err != nil {
				return err
			}
		}
	}
	for i := range ex.Employments {
		result := tx.Where("name = ?", ex.Employments[i].Name).FirstOrCreate(&ex.Employments[i])
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := audit.LogCreate(tx, r, "employments", ex.Employments[i].ID, ex.Employments[i]); err != nil {
				return err
			}
		}
	}
	for i := range ex.Programs {
		result := tx.Where("name = ?", ex.Programs[i].Name).FirstOrCreate(&ex.Programs[i])
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := audit.LogCreate(tx, r, "programs", ex.Programs[i].ID, ex.Programs[i]); err != nil {
				return err
			}
		}
	}
	if !sameIndustryIDs(before.Industries, ex.Industries) {
		if err := tx.Model(ex).Association("Industries").Replace(ex.Industries); err != nil {
			return err
		}
	}
	if !sameEmploymentIDs(before.Employments, ex.Employments) {
		if err := tx.Model(ex).Association("Employments").Replace(ex.Employments); err != nil {
			return err
		}
	}
	if !sameProgramIDs(before.Programs, ex.Programs) {
		if err := tx.Model(ex).Association("Programs").Replace(ex.Programs); err != nil {
			return err
		}
	}
	return nil
}

type EventroReviewField struct {
	Field    string `json:"field"`
	CMS      any    `json:"cms"`
	Eventro  any    `json:"eventro"`
	Conflict bool   `json:"conflict"`
}

type EventroReviewRow struct {
	Resource         string                         `json:"resource"`
	ID               uint                           `json:"id"`
	Name             string                         `json:"name"`
	Missing          []string                       `json:"missing"`
	Fields           []EventroReviewField           `json:"fields"`
	UnmappedProducts []models.EventroOrderedProduct `json:"unmappedProducts"`
	Notice           string                         `json:"notice,omitempty"`
}

func eventroReviewValues(resource string, entity any) map[string]json.RawMessage {
	if resource == "exhibitors" {
		return exhibitorSyncValues(*entity.(*models.Exhibitor))
	}
	return eventroStructValues(entity, eventroResourceSpecs[resource].fields)
}

func eventroReviewQuery(tx *gorm.DB, resource string) *gorm.DB {
	if resource == "exhibitors" {
		return tx.Preload("Industries").Preload("Employments").Preload("Programs")
	}
	return tx
}

// GetEventroSyncReview returns private source differences for one resource.
// @Summary Review Eventro changes and CMS corrections
// @Tags eventro
// @Produce json
// @Param resource query string true "CMS resource name"
// @Success 200 {array} EventroReviewRow
// @Security BearerAuth
// @Router /eventrosyncreview [get]
func GetEventroSyncReview(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	spec, ok := eventroResourceSpecs[resource]
	if !ok {
		http.Error(w, "invalid resource", 400)
		return
	}
	if !auth.HasPermission(auth.GetPermissionsFromContext(r), resource+".edit") {
		http.Error(w, "insufficient resource permissions", 403)
		return
	}
	slice := reflect.New(reflect.SliceOf(reflect.TypeOf(spec.newModel()).Elem()))
	if err := eventroReviewQuery(db.DB, resource).Order("id").Find(slice.Interface()).Error; err != nil {
		http.Error(w, "failed to load records", 500)
		return
	}
	var states []models.EventroSyncState
	if err := db.DB.Where("resource = ?", resource).Find(&states).Error; err != nil {
		http.Error(w, "failed to load history", 500)
		return
	}
	byID := map[uint]models.EventroSyncState{}
	for _, state := range states {
		byID[state.RecordID] = state
	}
	rows := []EventroReviewRow{}
	for i := 0; i < slice.Elem().Len(); i++ {
		entity := slice.Elem().Index(i).Addr().Interface()
		id := uint(slice.Elem().Index(i).FieldByName("ID").Uint())
		state := byID[id]
		values := eventroReviewValues(resource, entity)
		name := values["name"]
		if name == nil {
			name = values["description"]
		}
		var label string
		_ = json.Unmarshal(name, &label)
		row := EventroReviewRow{Resource: resource, ID: id, Name: label, Missing: []string{}, Fields: []EventroReviewField{}, UnmappedProducts: state.UnmappedProducts}
		if resource == "exhibitors" {
			for _, field := range []string{"companyWebsite", "about", "industries", "employments", "programs", "cities", "tier"} {
				if emptyEventroValue(values[field]) {
					row.Missing = append(row.Missing, field)
				}
			}
			ex := entity.(*models.Exhibitor)
			if ex.LogoSquaredUrl == nil && ex.LogoFreesizeUrl == nil && ex.LogoThumbnail == nil {
				row.Missing = append(row.Missing, "logo")
			}
		}
		if resource == "fairdates" && slice.Elem().Len() > 1 {
			row.Notice = "Multiple fair date configurations exist. Review which one should remain active; sync never deletes other configurations."
		}
		for field, source := range state.Source {
			if state.Overrides[field] || !sameEventroValue(values[field], source) {
				row.Fields = append(row.Fields, EventroReviewField{Field: field, CMS: values[field], Eventro: source, Conflict: state.Conflicts[field]})
			}
		}
		sort.Slice(row.Fields, func(i, j int) bool { return row.Fields[i].Field < row.Fields[j].Field })
		if len(row.Missing)+len(row.Fields)+len(row.UnmappedProducts) > 0 || row.Notice != "" {
			rows = append(rows, row)
		}
	}
	writeJSONResponse(w, http.StatusOK, rows)
}

type EventroReviewDecision struct {
	Resource string `json:"resource"`
	RecordID uint   `json:"recordId"`
	Field    string `json:"field"`
	Choice   string `json:"choice"`
	CMS      any    `json:"cms"`
	Eventro  any    `json:"eventro"`
}

var errEventroReviewStale = errors.New("values changed; reload the review")

// ResolveEventroSyncReview records field ownership with optimistic concurrency checks.
// @Summary Keep a CMS correction or resume Eventro synchronization for one field
// @Tags eventro
// @Accept json
// @Produce json
// @Param decision body EventroReviewDecision true "Reviewed values and decision"
// @Success 200 {object} map[string]bool
// @Failure 403 {string} string "Insufficient resource permissions"
// @Failure 409 {string} string "Values changed; reload the review"
// @Security BearerAuth
// @Router /eventrosyncreview [post]
func ResolveEventroSyncReview(w http.ResponseWriter, r *http.Request) {
	var decision EventroReviewDecision
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&decision) != nil || decision.RecordID == 0 || (decision.Choice != "cms" && decision.Choice != "eventro") {
		http.Error(w, "invalid review decision", 400)
		return
	}
	spec, ok := eventroResourceSpecs[decision.Resource]
	if !ok {
		http.Error(w, "invalid resource", 400)
		return
	}
	if !auth.HasPermission(auth.GetPermissionsFromContext(r), decision.Resource+".edit") {
		http.Error(w, "insufficient resource permissions", 403)
		return
	}
	allowed := false
	for _, field := range spec.fields {
		if field == decision.Field {
			allowed = true
		}
	}
	if !allowed {
		http.Error(w, "invalid field", 400)
		return
	}
	before, after := spec.newModel(), spec.newModel()
	err := updateWithAudit(r, decision.Resource, decision.RecordID, before, &after, func(tx *gorm.DB) error {
		if err := eventroReviewQuery(tx.Clauses(clause.Locking{Strength: "UPDATE"}), decision.Resource).First(before, decision.RecordID).Error; err != nil {
			return err
		}
		state, err := loadEventroSyncState(tx, decision.Resource, decision.RecordID)
		if err != nil {
			return err
		}
		value, known := state.Source[decision.Field]
		expectedEventro, _ := json.Marshal(decision.Eventro)
		expectedCMS, _ := json.Marshal(decision.CMS)
		if !known || !sameEventroValue(value, expectedEventro) || !sameEventroValue(eventroReviewValues(decision.Resource, before)[decision.Field], expectedCMS) {
			return errEventroReviewStale
		}
		stateJSON, _ := json.Marshal(state)
		var stateBefore models.EventroSyncState
		_ = json.Unmarshal(stateJSON, &stateBefore)
		recordJSON, _ := json.Marshal(before)
		if err := json.Unmarshal(recordJSON, after); err != nil {
			return err
		}
		if decision.Choice == "eventro" {
			if decision.Resource == "exhibitors" {
				ex := after.(*models.Exhibitor)
				if err := applyEventroReviewValue(ex, decision.Field, value); err != nil {
					return err
				}
				if err := tx.Model(ex).Omit(clause.Associations).Select("name", "tier", "company_website", "about", "logo_freesize_url", "logo_thumbnail", "cities").Updates(ex).Error; err != nil {
					return err
				}
				if err := persistEventroExhibitorCategories(tx, r, ex, *before.(*models.Exhibitor)); err != nil {
					return err
				}
			} else {
				if err := applyEventroStructValue(after, decision.Field, value); err != nil {
					return err
				}
				statement := &gorm.Statement{DB: tx}
				if err := statement.Parse(after); err != nil {
					return err
				}
				for _, field := range statement.Schema.Fields {
					if strings.Split(field.StructField.Tag.Get("json"), ",")[0] == decision.Field {
						v, _ := field.ValueOf(r.Context(), reflect.ValueOf(after))
						if err := tx.Model(after).Update(field.DBName, v).Error; err != nil {
							return err
						}
					}
				}
			}
			delete(state.Overrides, decision.Field)
		} else {
			if state.Overrides == nil {
				state.Overrides = map[string]bool{}
			}
			state.Overrides[decision.Field] = true
		}
		delete(state.Conflicts, decision.Field)
		return saveEventroSyncStateWithAudit(tx, r, stateBefore, &state)
	}, nil, spec.tags...)
	if err != nil {
		if errors.Is(err, errEventroReviewStale) {
			http.Error(w, err.Error(), 409)
		} else {
			http.Error(w, "failed to save review", 500)
		}
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func trimmedOptionalFormValue(r *http.Request, field string, current *string) *string {
	if _, present := r.MultipartForm.Value[field]; !present {
		return current
	}
	return nullIfEmpty(strings.TrimSpace(r.FormValue(field)))
}
