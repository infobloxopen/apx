package publisher

import "fmt"

// PublishErrorCode is a machine-readable identifier for publish failure modes.
type PublishErrorCode string

const (
	// ErrCodeVersionTaken means the requested version already exists with
	// different content. Choose a different version.
	ErrCodeVersionTaken PublishErrorCode = "VERSION_TAKEN"

	// ErrCodeLifecycleBlocked means the lifecycle state (e.g. sunset)
	// blocks new releases.
	ErrCodeLifecycleBlocked PublishErrorCode = "LIFECYCLE_BLOCKED"

	// ErrCodeLifecycleMismatch means the version prerelease tag does not
	// match the declared lifecycle (e.g. -alpha on a stable API).
	ErrCodeLifecycleMismatch PublishErrorCode = "LIFECYCLE_MISMATCH"

	// ErrCodeValidationFailed means schema validation (lint/breaking/policy)
	// did not pass.
	ErrCodeValidationFailed PublishErrorCode = "VALIDATION_FAILED"

	// ErrCodeGoPackageMismatch means a proto file's go_package option does
	// not match the derived import path.
	ErrCodeGoPackageMismatch PublishErrorCode = "GO_PACKAGE_MISMATCH"

	// ErrCodeGoModMismatch means an existing go.mod has a different module
	// path than expected.
	ErrCodeGoModMismatch PublishErrorCode = "GO_MOD_MISMATCH"

	// ErrCodeMergeConflict means the canonical repo has diverged and a
	// merge conflict occurred during subtree/copy publish.
	ErrCodeMergeConflict PublishErrorCode = "MERGE_CONFLICT"

	// ErrCodeCanonicalMoved means the canonical repo's HEAD has moved since
	// the prepare step.
	ErrCodeCanonicalMoved PublishErrorCode = "CANONICAL_MOVED"

	// ErrCodePolicyFailed means a canonical CI policy check failed.
	ErrCodePolicyFailed PublishErrorCode = "POLICY_FAILED"

	// ErrCodePackagePublishFailed means language package publication failed
	// after the canonical tag was created.
	ErrCodePackagePublishFailed PublishErrorCode = "PACKAGE_PUBLISH_FAILED"

	// ErrCodeCatalogUpdateFailed means the catalog update failed after
	// canonical release.
	ErrCodeCatalogUpdateFailed PublishErrorCode = "CATALOG_UPDATE_FAILED"

	// ErrCodeNotGitRepo means the current directory is not a git repository.
	ErrCodeNotGitRepo PublishErrorCode = "NOT_GIT_REPO"

	// ErrCodeInvalidVersion means the version string is not valid semver.
	ErrCodeInvalidVersion PublishErrorCode = "INVALID_VERSION"

	// ErrCodeMissingConfig means apx.yaml could not be loaded.
	ErrCodeMissingConfig PublishErrorCode = "MISSING_CONFIG"

	// ErrCodeSubtreeFailed means the git subtree split failed.
	ErrCodeSubtreeFailed PublishErrorCode = "SUBTREE_FAILED"

	// ErrCodePushFailed means the git push to canonical repo failed.
	ErrCodePushFailed PublishErrorCode = "PUSH_FAILED"
)

// PublishError is a structured error with a machine-readable code,
// human-readable message, and optional recovery hint.
type PublishError struct {
	Code    PublishErrorCode `json:"code"`
	Message string           `json:"message"`
	Hint    string           `json:"hint,omitempty"`
}

func (e *PublishError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("[%s] %s (hint: %s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewPublishError creates a new PublishError.
func NewPublishError(code PublishErrorCode, message string) *PublishError {
	return &PublishError{Code: code, Message: message}
}

// WithHint returns a copy of the error with a recovery hint attached.
func (e *PublishError) WithHint(hint string) *PublishError {
	return &PublishError{Code: e.Code, Message: e.Message, Hint: hint}
}

// ErrorCodeDescriptions maps each error code to a brief description
// suitable for documentation and CLI help.
var ErrorCodeDescriptions = map[PublishErrorCode]string{
	ErrCodeVersionTaken:         "Version already exists with different content",
	ErrCodeLifecycleBlocked:     "Lifecycle state blocks new releases",
	ErrCodeLifecycleMismatch:    "Version prerelease tag conflicts with lifecycle",
	ErrCodeValidationFailed:     "Schema validation did not pass",
	ErrCodeGoPackageMismatch:    "Proto go_package does not match derived import",
	ErrCodeGoModMismatch:        "go.mod module directive mismatch",
	ErrCodeMergeConflict:        "Merge conflict in canonical repo",
	ErrCodeCanonicalMoved:       "Canonical repo HEAD has moved since prepare",
	ErrCodePolicyFailed:         "Canonical CI policy check failed",
	ErrCodePackagePublishFailed: "Package publication failed after canonical tag",
	ErrCodeCatalogUpdateFailed:  "Catalog update failed after release",
	ErrCodeNotGitRepo:           "Not inside a git repository",
	ErrCodeInvalidVersion:       "Version is not valid semver",
	ErrCodeMissingConfig:        "apx.yaml not found or invalid",
	ErrCodeSubtreeFailed:        "Git subtree split failed",
	ErrCodePushFailed:           "Git push to canonical repo failed",
}
