package utils

import "testing"

func TestStringPtrReturnsNilForEmptyString(t *testing.T) {
	if got := StringPtr(""); got != nil {
		t.Fatalf("expected StringPtr(\"\") to return nil, got %#v", got)
	}
}

func TestStringPtrReturnsPointerForNonEmptyString(t *testing.T) {
	got := StringPtr("hello")
	if got == nil {
		t.Fatal("expected StringPtr(\"hello\") to return non-nil pointer")
		return
	}
	if *got != "hello" {
		t.Fatalf("expected pointed-to value to be \"hello\", got %q", *got)
	}
}

func TestToSnakeCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"foo", "foo"},
		{"FooBar", "foo_bar"},
		{"fooBarBaz", "foo_bar_baz"},
		{"alreadysnake", "alreadysnake"},
		{"ID", "i_d"},
		{"userID", "user_i_d"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := ToSnakeCase(tc.in)
			if got != tc.want {
				t.Fatalf("ToSnakeCase(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
