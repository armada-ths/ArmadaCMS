package utils

import "github.com/microcosm-cc/bluemonday"

var eventDescriptionPolicy = newEventDescriptionPolicy()

func newEventDescriptionPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u", "s",
		"ul", "ol", "li", "blockquote", "a", "span",
	)
	policy.AllowStandardURLs()
	policy.AllowAttrs("href", "title").OnElements("a")
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)

	return policy
}

// SanitizeEventDescription removes executable markup while preserving the
// small formatting subset supported by event descriptions.
func SanitizeEventDescription(description *string) *string {
	if description == nil {
		return nil
	}

	sanitized := eventDescriptionPolicy.Sanitize(*description)
	return &sanitized
}
