package translations

import (
	"ArmadaCMS/main/utils"
	_ "embed"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

//go:embed prompt.md
var Prompt string

//go:embed glossary.json
var Glossary string
var PromptVersion = Hash(Prompt)
var GlossaryVersion = Hash(json.RawMessage(Glossary))

type Term struct {
	English string   `json:"english"`
	Swedish string   `json:"swedish"`
	Context string   `json:"context"`
	Stems   []string `json:"stems"`
}
type Policy struct {
	Protected []string `json:"protected"`
	Terms     []Term   `json:"terms"`
}

var protectedTokens = regexp.MustCompile("https?://[^\\s<>\\\"]+|[\\w.+-]+@[\\w.-]+\\.[A-Za-z]{2,}|\\{[^{}\\r\\n]+\\}|%[0-9.$+-]*[sdif]|[0-9]+(?:[.,:/-][0-9]+)*|`[^`\\r\\n]+`")
var markdownStructure = regexp.MustCompile("(?m)^\\s*(#{1,6} |[-*+] |[0-9]+[.)] |>{1,} |```[^\\r\\n]*|~~~[^\\r\\n]*)")

func matches(pattern *regexp.Regexp, s string) []string {
	v := pattern.FindAllString(s, -1)
	slices.Sort(v)
	return v
}
func htmlShape(s string) ([]string, error) {
	tokens := []string{}
	z := html.NewTokenizer(strings.NewReader(s))
	for {
		switch z.Next() {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			return tokens, nil
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			attrs := []string{}
			for _, a := range t.Attr {
				if a.Key == "title" || a.Key == "alt" {
					// These values are translatable, but the attributes must remain present.
					attrs = append(attrs, a.Key+fmt.Sprint(strings.TrimSpace(a.Val) != ""))
				} else {
					attrs = append(attrs, a.Key+"="+a.Val)
				}
			}
			slices.Sort(attrs)
			tokens = append(tokens, "<"+t.Data+encode(attrs)+">")
		case html.EndTagToken:
			tokens = append(tokens, "</"+z.Token().Data+">")
		}
	}
}

// Names must be words rather than substrings (THS is not the end of MONTHS).
// Swedish genitive -s keeps the proper name intact.
func protectedOccurrences(value, name string) int {
	count, offset := 0, 0
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }
	for offset < len(value) {
		index := strings.Index(value[offset:], name)
		if index < 0 {
			break
		}
		index += offset
		offset = index + len(name)
		if index > 0 {
			before, _ := utf8.DecodeLastRuneInString(value[:index])
			if word(before) {
				continue
			}
		}
		suffix := strings.TrimPrefix(value[offset:], "s")
		if len(suffix) > 0 {
			after, _ := utf8.DecodeRuneInString(suffix)
			if word(after) {
				continue
			}
		}
		count++
	}
	return count
}
func ValidateText(source Source, f Field, value, glossary string) ([]string, error) {
	original := source.Fields[f.ID]
	nordic := regexp.MustCompile(`(?i)\bnordic\b`).MatchString(original)
	scandinavian := regexp.MustCompile(`(?i)\bscandinavi(a|an)`).MatchString(original)
	swedishNordic := regexp.MustCompile(`(?i)nord(en|isk)`).MatchString(value)
	swedishScandinavian := regexp.MustCompile(`(?i)skandinav`).MatchString(value)
	if (nordic && !scandinavian && swedishScandinavian && !swedishNordic) || (scandinavian && !nordic && swedishNordic && !swedishScandinavian) {
		return nil, fmt.Errorf("geographic scope changed")
	}

	if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || len(value) > len(original)*8+4096 {
		return nil, fmt.Errorf("empty, invalid or oversized translated text")
	}
	if !slices.Equal(matches(protectedTokens, original), matches(protectedTokens, value)) {
		return nil, fmt.Errorf("protected links, numbers, placeholders or code changed")
	}
	var policy Policy
	if json.Unmarshal([]byte(glossary), &policy) != nil {
		return nil, fmt.Errorf("invalid glossary")
	}
	if source.Resource == "profiles" {
		policy.Protected = append(policy.Protected, source.Label)
	}
	for _, name := range policy.Protected {
		if name != "" && protectedOccurrences(original, name) != protectedOccurrences(value, name) {
			return nil, fmt.Errorf("protected name changed")
		}
	}
	if f.Format == "html" {
		safe := utils.SanitizeEventDescription(&value)
		if safe == nil || *safe != value {
			return nil, fmt.Errorf("unsafe HTML")
		}
	}
	if f.Format == "html" || f.Format == "markdown" {
		a, err := htmlShape(original)
		if err != nil {
			return nil, err
		}
		b, err := htmlShape(value)
		if err != nil || !slices.Equal(a, b) {
			return nil, fmt.Errorf("HTML structure changed")
		}
	}
	if f.Format == "markdown" && (!slices.Equal(matches(markdownStructure, original), matches(markdownStructure, value)) || !slices.Equal(markdownShape(original), markdownShape(value))) {
		return nil, fmt.Errorf("markdown structure changed")
	}
	warnings := []string{}
	for _, term := range policy.Terms {
		if strings.Contains(strings.ToLower(original), strings.ToLower(term.English)) {
			found := false
			for _, stem := range term.Stems {
				found = found || strings.Contains(strings.ToLower(value), strings.ToLower(stem))
			}
			if !found {
				warnings = append(warnings, "Check terminology: "+term.English+" / "+term.Swedish+" ("+term.Context+")")
			}
		}
	}
	return warnings, nil
}
