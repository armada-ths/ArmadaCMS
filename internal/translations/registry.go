package translations

import (
	"ArmadaCMS/main/models"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"strings"
)

type Field struct {
	ID     string `json:"id"`
	Source string `json:"-"`
	Target string `json:"-"`
	Format string `json:"format"`
}
type Resource struct {
	ID     string     `json:"id"`
	Label  string     `json:"label"`
	Fields []Field    `json:"fields"`
	AI     bool       `json:"aiAllowed"`
	Tags   []string   `json:"-"`
	Model  func() any `json:"-"`
	Title  string     `json:"-"`
}

func field(source, format string) Field {
	return Field{ID: source, Source: source, Target: source + "Sv", Format: format}
}

var Resources = []Resource{
	{"highlightcards", "Highlight cards", []Field{field("Title", "text"), field("Subtitle", "text"), field("Description", "text"), field("LinkText", "text")}, true, []string{"highlight-cards"}, func() any { return &models.HighlightCard{} }, "Title"},
	{"blogposts", "Blog posts", []Field{field("Title", "text"), field("Text", "markdown")}, true, []string{"blog-posts"}, func() any { return &models.Blogpost{} }, "Title"},
	{"events", "Events", []Field{field("Name", "text"), field("Description", "html"), field("Location", "text"), field("OpenTo", "text"), field("Food", "text"), field("Fee", "text")}, true, []string{"events"}, func() any { return &models.Event{} }, "Name"},
	{"exhibitors", "Exhibitor descriptions (manual only)", []Field{field("About", "text")}, false, []string{"exhibitors"}, func() any { return &models.Exhibitor{} }, "Name"},
	{"employments", "Employment labels", []Field{field("Name", "text")}, true, []string{"employments", "exhibitors"}, func() any { return &models.Employment{} }, "Name"},
	{"industries", "Industry labels", []Field{field("Name", "text")}, true, []string{"industries", "exhibitors"}, func() any { return &models.Industry{} }, "Name"},
	{"programs", "Study program labels", []Field{field("Name", "text")}, true, []string{"programs", "exhibitors"}, func() any { return &models.Program{} }, "Name"},
	{"teams", "Teams", []Field{field("TeamName", "text")}, true, []string{"organization", "recruitment"}, func() any { return &models.Team{} }, "TeamName"},
	{"profiles", "Profile role titles", []Field{field("Title", "text")}, true, []string{"organization"}, func() any { return &models.Profile{} }, "Name"},
	{"recruitmentperiods", "Recruitment periods", []Field{field("Name", "text")}, true, []string{"recruitment"}, func() any { return &models.RecruitmentPeriod{} }, "Name"},
	{"recruitmentroles", "Recruitment roles", []Field{field("Name", "text"), field("Description", "text")}, true, []string{"recruitment"}, func() any { return &models.RecruitmentRole{} }, "Name"},
	{"fairdates", "Fair dates", []Field{field("Description", "text")}, true, []string{"dates"}, func() any { return &models.FairDateConfig{} }, "Description"},
	{"timeline-eras", "History eras", []Field{field("Title", "text")}, true, []string{"timeline-entries"}, func() any { return &models.TimelineEra{} }, "Title"},
	{"timeline-entries", "History entries", []Field{field("Title", "text"), field("Body", "markdown")}, true, []string{"timeline-entries"}, func() any { return &models.TimelineEntry{} }, "Title"},
	{"photoevents", "Photo events", []Field{field("Name", "text"), field("Description", "text")}, true, nil, func() any { return &models.PhotoEvent{} }, "Name"},
}

func Find(id string) (Resource, error) {
	for _, r := range Resources {
		if r.ID == id {
			return r, nil
		}
	}
	return Resource{}, fmt.Errorf("unknown translation resource")
}
func (r Resource) Field(id string) (Field, error) {
	for _, f := range r.Fields {
		if f.ID == id {
			return f, nil
		}
	}
	return Field{}, fmt.Errorf("unknown translation field")
}
func textValue(model any, name string) *string {
	v := reflect.ValueOf(model).Elem().FieldByName(name)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	s := v.String()
	return &s
}
func Blank(s *string) bool    { return s == nil || strings.TrimSpace(*s) == "" }
func Equal(a, b *string) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func (r Resource) Load(tx *gorm.DB, id uint64, lock bool) (any, error) {
	m := r.Model()
	q := tx
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(m, id).Error
	return m, err
}

type Source struct {
	Resource string             `json:"resource"`
	RecordID uint64             `json:"recordId"`
	Label    string             `json:"label"`
	Fields   map[string]string  `json:"fields"`
	Formats  map[string]string  `json:"formats"`
	Targets  map[string]*string `json:"targets"`
}

func (r Resource) Snapshot(m any, id uint64) Source {
	s := Source{r.ID, id, "", map[string]string{}, map[string]string{}, map[string]*string{}}
	if title := textValue(m, r.Title); title != nil {
		s.Label = *title
	}
	for _, f := range r.Fields {
		if v := textValue(m, f.Source); v != nil {
			s.Fields[f.ID] = *v
		}
		s.Formats[f.ID] = f.Format
		s.Targets[f.ID] = textValue(m, f.Target)
	}
	return s
}
func Hash(v any) string { b, _ := json.Marshal(v); return fmt.Sprintf("%x", sha256.Sum256(b)) }
func (s Source) Fingerprint() string {
	return Hash(struct {
		Label  string
		Fields map[string]string
	}{s.Label, s.Fields})
}
func (r Resource) Columns(tx *gorm.DB, values map[string]string) (map[string]any, error) {
	statement := &gorm.Statement{DB: tx}
	if err := statement.Parse(r.Model()); err != nil {
		return nil, err
	}
	updates := map[string]any{}
	for id, text := range values {
		f, err := r.Field(id)
		if err != nil {
			return nil, err
		}
		column := statement.Schema.LookUpField(f.Target)
		if column == nil {
			return nil, fmt.Errorf("missing translation column")
		}
		var value *string
		if strings.TrimSpace(text) != "" {
			s := strings.TrimSpace(text)
			value = &s
		}
		updates[column.DBName] = value
	}
	return updates, nil
}
