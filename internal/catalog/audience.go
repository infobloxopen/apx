package catalog

import "fmt"

// The audiences of an API. The audience says who can call the API: public is
// for callers outside the organization, internal is for callers inside it.
const (
	AudiencePublic   = "public"
	AudienceInternal = "internal"
)

// ValidateAudience accepts "public", "internal" and the empty string. The
// audience is optional: an empty value means that the module does not state
// one, and no default is applied.
func ValidateAudience(audience string) error {
	switch audience {
	case "", AudiencePublic, AudienceInternal:
		return nil
	}
	return fmt.Errorf("invalid audience %q (expected %s or %s)", audience, AudiencePublic, AudienceInternal)
}
