package utils

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ApplySearch adds a case-insensitive literal substring search over the supplied
// columns. Columns must be server-defined; qualified names support joined lists.
// Grouping the alternatives preserves other filters and visibility restrictions.
func ApplySearch(query *gorm.DB, term string, columns ...string) *gorm.DB {
	term = strings.TrimSpace(term)
	if term == "" || len(columns) == 0 {
		return query
	}
	like := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(term) + "%"
	alternatives := make([]clause.Expression, 0, len(columns))
	for _, column := range columns {
		table, name, qualified := strings.Cut(column, ".")
		field := clause.Column{Name: column}
		if qualified {
			field = clause.Column{Table: table, Name: name}
		}
		alternatives = append(alternatives, clause.Expr{
			SQL:  `CAST(? AS TEXT) ILIKE ? ESCAPE '!'`,
			Vars: []any{field, like},
		})
	}
	return query.Where(clause.Or(alternatives...))
}
