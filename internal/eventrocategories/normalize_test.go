package eventrocategories

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
)

func TestEveryCSVMapping(t *testing.T) {
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(mappingCSV, "\ufeff"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		t.Run(row[0]+"/"+row[1], func(t *testing.T) {
			kind, expected := CategoryType(row[0]), []string{row[2]}
			for _, input := range []string{row[1], "  " + strings.ToUpper(row[1]) + "  ", row[2]} {
				actual := Normalize(kind, []string{input})
				if !reflect.DeepEqual(actual, expected) {
					t.Fatalf("%q: got %v, want %v", input, actual, expected)
				}
			}
		})
	}
}

func TestDeduplicationUnknownsAndTypeIsolation(t *testing.T) {
	cases := []struct {
		kind            CategoryType
		input, expected []string
	}{
		{Industry, []string{" advisory ", "consulting", "CONSULTING & ADVISORY", "", " New  industry "}, []string{"Consulting & Advisory", "New industry"}},
		{Employment, []string{"Master Thesis ", "Masters Thesis", "Master's thesis", "Thesis work"}, []string{"Master's thesis", "Thesis projects"}},
		{Industry, []string{"energy"}, []string{"Energy"}},
		{FieldOfStudy, []string{"energy", "electricity", "Computer Engineering", "Computer Science (Artificial Intelligence, Data Science, Software Engineering)"}, []string{"Computer Science", "Electrical Engineering", "Energy Engineering"}},
		{Industry, nil, []string{}},
	}
	for _, c := range cases {
		original := append([]string{}, c.input...)
		actual := Normalize(c.kind, c.input)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Fatalf("%s: got %v, want %v", c.kind, actual, c.expected)
		}
		if strings.Join(original, "|") != strings.Join(c.input, "|") {
			t.Fatal("source labels modified")
		}
	}
}

func TestInvalidMapping(t *testing.T) {
	header := "Type,Original category,Normalized category,Review\n"
	for _, data := range []string{
		"wrong header\n",
		header + "Unknown,a,b,\n",
		header + "Industry,,b,\n",
		header + "Industry,a,,\n",
		header + "Industry,a,First,\nIndustry,A,Second,\n",
	} {
		if _, err := loadMappings(data); err == nil {
			t.Fatalf("invalid mapping accepted: %q", data)
		}
	}
}
