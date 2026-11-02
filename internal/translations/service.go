package translations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"reflect"
	"slices"
	"strings"
	"time"
)

var ErrConflict = errors.New("translation changed; reload the record before reviewing")
var ErrDisabled = errors.New("AI translation generation is disabled")

type Service struct {
	DB     *gorm.DB
	Config Config
	Now    func() time.Time
}

func New(database *gorm.DB) *Service {
	return &Service{database.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), ConfigFromEnv(), time.Now}
}
func encode(v any) string { b, _ := json.Marshal(v); return string(b) }

type FieldView struct {
	ID          string      `json:"id"`
	Format      string      `json:"format"`
	Source      string      `json:"source"`
	Published   *string     `json:"published"`
	Status      string      `json:"status"`
	Suggestion  *Suggestion `json:"suggestion,omitempty"`
	Warnings    []string    `json:"warnings"`
	OlderPolicy bool        `json:"olderPolicy"`
}
type Record struct {
	ID          uint64      `json:"id"`
	Label       string      `json:"label"`
	Fingerprint string      `json:"fingerprint"`
	Fields      []FieldView `json:"fields"`
	Job         *Job        `json:"job,omitempty"`
}
type Page struct {
	Records []Record `json:"records"`
	Total   int      `json:"total"`
}

func latestSuggestions(tx *gorm.DB, resource string, id uint64) (map[string]Suggestion, error) {
	var items []Suggestion
	q := tx.Where("resource = ?", resource)
	if id > 0 {
		q = q.Where("record_id = ?", id)
	}
	err := q.Select("DISTINCT ON (record_id, field_id) *").Order("record_id, field_id, id DESC").Find(&items).Error
	latest := map[string]Suggestion{}
	for _, item := range items {
		key := fmt.Sprintf("%d/%s", item.RecordID, item.FieldID)
		if _, ok := latest[key]; !ok {
			latest[key] = item
		}
	}
	return latest, err
}
func buildRecord(r Resource, m any, id uint64, latest map[string]Suggestion, jobs map[uint64]Job) Record {
	source := r.Snapshot(m, id)
	record := Record{ID: id, Label: source.Label, Fingerprint: source.Fingerprint(), Fields: []FieldView{}}
	if job, ok := jobs[id]; ok {
		if job.Fingerprint != record.Fingerprint && (job.Status == "queued" || job.Status == "budget_wait" || job.Status == "running") {
			job.Outdated = true
		}
		record.Job = &job
	}
	for _, field := range r.Fields {
		view := FieldView{ID: field.ID, Format: field.Format, Source: source.Fields[field.ID], Published: source.Targets[field.ID], Status: "translated", Warnings: []string{}}
		if Blank(view.Published) {
			view.Status = "missing"
		}
		if strings.TrimSpace(view.Source) == "" {
			view.Status = "empty_source"
		}
		if suggestion, ok := latest[fmt.Sprintf("%d/%s", id, field.ID)]; ok {
			view.Suggestion = &suggestion
			view.OlderPolicy = suggestion.PromptVersion != "" && (suggestion.PromptVersion != PromptVersion || suggestion.GlossaryVersion != GlossaryVersion)
			_ = json.Unmarshal([]byte(suggestion.Warnings), &view.Warnings)
			switch suggestion.Status {
			case "draft":
				view.Status = "review"
				if suggestion.SourceFingerprint != record.Fingerprint || !Equal(suggestion.TargetBefore, view.Published) {
					view.Status = "stale"
				}
			case "rejected":
				if Blank(view.Published) {
					view.Status = "rejected"
				}
			case "approved":
				if Equal(suggestion.PublishedText, view.Published) && suggestion.SourceFingerprint != record.Fingerprint {
					view.Status = "needs_update"
				}
			}
		}
		record.Fields = append(record.Fields, view)
	}
	return record
}
func (s *Service) List(ctx context.Context, r Resource, search, status string, page, size int) (Page, error) {
	latest, err := latestSuggestions(s.DB.WithContext(ctx), r.ID, 0)
	if err != nil {
		return Page{}, err
	}
	var jobRows []Job
	if err = s.DB.WithContext(ctx).Select("DISTINCT ON (record_id) id, resource, record_id, status, error, fingerprint, model, prompt_version, glossary_version, not_before, created_at").Where("resource = ?", r.ID).Order("record_id, id DESC").Find(&jobRows).Error; err != nil {
		return Page{}, err
	}
	jobs := map[uint64]Job{}
	for _, job := range jobRows {
		if _, exists := jobs[job.RecordID]; !exists {
			jobs[job.RecordID] = job
		}
	}
	statement := &gorm.Statement{DB: s.DB}
	if err = statement.Parse(r.Model()); err != nil {
		return Page{}, err
	}
	cols := []string{"id", statement.Schema.LookUpField(r.Title).DBName}
	for _, f := range r.Fields {
		cols = append(cols, statement.Schema.LookUpField(f.Source).DBName, statement.Schema.LookUpField(f.Target).DBName)
	}
	slices.Sort(cols)
	cols = slices.Compact(cols)
	slice := reflect.New(reflect.SliceOf(reflect.TypeOf(r.Model()).Elem()))
	if err = s.DB.WithContext(ctx).Model(r.Model()).Select(cols).Order("id ASC").Find(slice.Interface()).Error; err != nil {
		return Page{}, err
	}
	out := Page{Records: []Record{}}
	for i := 0; i < slice.Elem().Len(); i++ {
		m := slice.Elem().Index(i).Addr().Interface()
		id := reflect.ValueOf(m).Elem().FieldByName("ID").Uint()
		record := buildRecord(r, m, id, latest, jobs)
		found := status == "all"
		searchable := record.Label
		for _, f := range record.Fields {
			searchable += " " + f.Source
			if f.Published != nil {
				searchable += " " + *f.Published
			}
			if status == f.Status || (status == "missing" && ((Blank(f.Published) && strings.TrimSpace(f.Source) != "") || f.Status == "stale" || f.Status == "needs_update")) {
				found = true
			}
		}
		if record.Job != nil && (status == record.Job.Status || (status == "stale" && record.Job.Outdated)) {
			found = true
		}
		if !found || !strings.Contains(strings.ToLower(searchable), strings.ToLower(search)) {
			continue
		}
		if out.Total >= page*size && len(out.Records) < size {
			out.Records = append(out.Records, record)
		}
		out.Total++
	}
	return out, nil
}
func (s *Service) Record(ctx context.Context, r Resource, id uint64) (Record, error) {
	m, err := r.Load(s.DB.WithContext(ctx), id, false)
	if err != nil {
		return Record{}, err
	}
	latest, err := latestSuggestions(s.DB.WithContext(ctx), r.ID, id)
	if err != nil {
		return Record{}, err
	}
	jobs := map[uint64]Job{}
	var job Job
	if err = s.DB.WithContext(ctx).Where("resource = ? AND record_id = ?", r.ID, id).Order("id DESC").First(&job).Error; err == nil {
		jobs[id] = job
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Record{}, err
	}
	return buildRecord(r, m, id, latest, jobs), nil
}
func (s *Service) Generate(ctx context.Context, r Resource, ids []uint64, user int, force bool) ([]Job, error) {
	if !s.Config.Enabled {
		return nil, ErrDisabled
	}
	if !r.AI {
		return nil, fmt.Errorf("exhibitor-owned descriptions are excluded from AI generation")
	}
	if len(ids) == 0 || len(ids) > s.Config.MaxRecords {
		return nil, fmt.Errorf("select between 1 and %d records", s.Config.MaxRecords)
	}
	seen := map[uint64]bool{}
	for _, id := range ids {
		if id == 0 || seen[id] {
			return nil, fmt.Errorf("invalid or duplicate record ID")
		}
		seen[id] = true
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	jobs := []Job{}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			m, err := r.Load(tx, id, true)
			if err != nil {
				return err
			}
			source := r.Snapshot(m, id)
			latest, err := latestSuggestions(tx, r.ID, id)
			if err != nil {
				return err
			}
			fields := []string{}
			for _, f := range r.Fields {
				if strings.TrimSpace(source.Fields[f.ID]) == "" {
					continue
				}
				previous, exists := latest[fmt.Sprintf("%d/%s", id, f.ID)]
				missing := Blank(source.Targets[f.ID])
				outdated := exists && previous.JobID != nil && previous.Status == "approved" && Equal(previous.PublishedText, source.Targets[f.ID]) && previous.SourceFingerprint != source.Fingerprint()
				if !missing && !outdated {
					continue
				}
				if exists && !force && previous.SourceFingerprint == source.Fingerprint() && (previous.Status == "draft" || previous.Status == "rejected") {
					continue
				}
				fields = append(fields, f.ID)
			}
			if len(fields) == 0 {
				continue
			}
			key := Hash(struct {
				Source           Source
				Fields           []string
				Prompt, Glossary string
			}{source, fields, PromptVersion, GlossaryVersion})
			var existing Job
			err = tx.Where("dedup_key = ? AND status IN ?", key, []string{"queued", "running", "budget_wait"}).First(&existing).Error
			if err == nil {
				jobs = append(jobs, existing)
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			job := Job{Resource: r.ID, RecordID: id, RequestedBy: user, Status: "queued", DedupKey: key, SourceJSON: encode(source), FieldIDs: encode(fields), Fingerprint: source.Fingerprint(), Model: s.Config.Model, Prompt: Prompt, Glossary: Glossary, PromptVersion: PromptVersion, GlossaryVersion: GlossaryVersion, NotBefore: s.Now().UTC()}
			if err = tx.Create(&job).Error; err != nil {
				return err
			}
			jobs = append(jobs, job)
		}
		return nil
	})
	return jobs, err
}

type ReviewField struct {
	FieldID      string  `json:"fieldId"`
	SuggestionID uint64  `json:"suggestionId"`
	Version      int     `json:"version"`
	Text         string  `json:"text"`
	TargetBefore *string `json:"targetBefore"`
}
type Review struct {
	Action      string        `json:"action"`
	Fingerprint string        `json:"fingerprint"`
	Fields      []ReviewField `json:"fields"`
}
type Publisher func(tx *gorm.DB, before, after any, updates map[string]any) error

func (s *Service) Review(ctx context.Context, r Resource, id uint64, input Review, user int, publish Publisher) error {
	if input.Action != "approve" && input.Action != "reject" && input.Action != "draft" {
		return fmt.Errorf("invalid review action")
	}
	if len(input.Fields) == 0 || len(input.Fields) > len(r.Fields) {
		return fmt.Errorf("select fields to review")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		before, err := r.Load(tx, id, true)
		if err != nil {
			return err
		}
		source := r.Snapshot(before, id)
		if input.Fingerprint != source.Fingerprint() {
			return ErrConflict
		}
		values := map[string]string{}
		suggestions := []Suggestion{}
		seen := map[string]bool{}
		for _, selection := range input.Fields {
			f, err := r.Field(selection.FieldID)
			if err != nil || seen[f.ID] {
				return fmt.Errorf("invalid or duplicate field")
			}
			seen[f.ID] = true
			if !Equal(selection.TargetBefore, source.Targets[f.ID]) {
				return ErrConflict
			}
			var suggestion Suggestion
			if selection.SuggestionID > 0 {
				if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND resource = ? AND record_id = ? AND field_id = ?", selection.SuggestionID, r.ID, id, f.ID).First(&suggestion).Error; err != nil {
					return ErrConflict
				}
				if suggestion.Version != selection.Version || suggestion.Status != "draft" || suggestion.SourceFingerprint != source.Fingerprint() || !Equal(suggestion.TargetBefore, source.Targets[f.ID]) {
					return ErrConflict
				}
			} else {
				var concurrent []Suggestion
				if err = tx.Where("resource = ? AND record_id = ? AND field_id = ? AND status = 'draft'", r.ID, id, f.ID).Find(&concurrent).Error; err != nil {
					return err
				}
				for _, other := range concurrent {
					if other.SourceFingerprint == source.Fingerprint() && Equal(other.TargetBefore, source.Targets[f.ID]) {
						return ErrConflict
					}
				}
				if input.Action == "reject" {
					return fmt.Errorf("no suggestion to reject")
				}
				suggestion = Suggestion{Resource: r.ID, RecordID: id, FieldID: f.ID, SourceFingerprint: source.Fingerprint(), TargetBefore: source.Targets[f.ID], Status: "draft", Warnings: "[]", PromptVersion: PromptVersion, GlossaryVersion: GlossaryVersion}
				if err = tx.Model(&Suggestion{}).Where("resource = ? AND record_id = ? AND field_id = ? AND status = 'draft'", r.ID, id, f.ID).Updates(map[string]any{"status": "superseded", "version": gorm.Expr("version + 1")}).Error; err != nil {
					return err
				}
			}
			value := strings.TrimSpace(selection.Text)
			if input.Action == "approve" && value != "" {
				warnings, err := ValidateText(source, f, value, Glossary)
				if err != nil {
					return fmt.Errorf("%s: %w", f.ID, err)
				}
				suggestion.Warnings = encode(warnings)
			}
			suggestion.Draft = value
			suggestion.Version++
			if input.Action == "approve" {
				suggestion.Status = "approved"
				var published *string
				if value != "" {
					published = &value
				}
				suggestion.PublishedText = published
				values[f.ID] = value
			}
			if input.Action == "reject" {
				suggestion.Status = "rejected"
			}
			if input.Action != "draft" {
				now := s.Now().UTC()
				suggestion.ReviewedBy = &user
				suggestion.ReviewedAt = &now
			}
			suggestions = append(suggestions, suggestion)
		}
		if input.Action == "approve" {
			updates, err := r.Columns(tx, values)
			if err != nil {
				return err
			}
			after := r.Model()
			if err = publish(tx, before, after, updates); err != nil {
				return err
			}
		}
		for i := range suggestions {
			if err = tx.Save(&suggestions[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Service) Cancel(ctx context.Context, r Resource, id uint64) error {
	return s.DB.WithContext(ctx).Model(&Job{}).Where("resource = ? AND record_id = ? AND status IN ?", r.ID, id, []string{"queued", "budget_wait"}).Updates(map[string]any{"status": "cancelled", "finished_at": s.Now().UTC()}).Error
}
