// Package materials defines teacher-authored learner materials: storage names, upload limits, and the rich-text sanitization rule.
// It has no HTTP or persistence side effects.
package materials

import (
	"errors"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

const (
	CollectionName   = "learner_materials"
	AssignmentField  = "assignment"
	TitleField       = "title"
	BodyField        = "body"
	AttachmentsField = "attachments"

	TitleMaxLength = 200
	BodyMaxBytes   = 200 << 10
	MaxFiles       = 10
	MaxFileBytes   = 10 << 20
	// A create request carries every file plus the title and body in one multipart form.
	MaxRequestBytes = MaxFiles*MaxFileBytes + BodyMaxBytes + (64 << 10)
)

// AllowedMimeTypes lists the image and PDF types that browsers render inline.
var AllowedMimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif", "application/pdf"}

var ErrEmpty = errors.New("material requires a title and a body or an attachment")

var (
	bodyPolicy = newBodyPolicy()
	textPolicy = bluemonday.StrictPolicy()
)

func newBodyPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	// Browser contentEditable wraps each typed line in a div.
	policy.AllowElements("div", "p", "br", "strong", "b", "em", "i", "u", "s", "h3", "h4", "ul", "ol", "li", "blockquote")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowURLSchemes("http", "https", "mailto")
	policy.RequireParseableURLs(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	policy.AddSpaceWhenStrippingTag(true)
	return policy
}

// SanitizeBody keeps only the formatting the teacher editor produces and removes scripts, styles, and unsafe links.
// A body without visible text becomes empty.
func SanitizeBody(html string) string {
	clean := strings.TrimSpace(bodyPolicy.Sanitize(html))
	if strings.TrimSpace(strings.ReplaceAll(textPolicy.Sanitize(clean), "&nbsp;", "")) == "" {
		return ""
	}
	return clean
}

// Validate checks the rule that a material has a title and at least one content part.
func Validate(title, sanitizedBody string, fileCount int) error {
	if strings.TrimSpace(title) == "" || (sanitizedBody == "" && fileCount == 0) {
		return ErrEmpty
	}
	return nil
}
