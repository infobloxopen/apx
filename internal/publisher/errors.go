package publisher

import "fmt"

// ReleaseErrorCode is a machine-readable identifier for release failure modes.
type ReleaseErrorCode string

const (
	// ErrCodeVersionTaken means the requested version already exists with
	// different content. Choose a different version.
	ErrCodeVersionTaken ReleaseErrorCode = "VERSION_TAKEN"

	// ErrCodeLifecycleBlocked means the lifecycle state (e.g. sunset)
	// blocks new releases.
	ErrCodeLifecycleBlocked ReleaseErrorCode = "LIFECYCLE_BLOCKED"

	// ErrCodeLifecycleMismatch means the version prerelease tag does not
	// match the declared lifecycle (e.g. -alpha on a stable API).
	ErrCodeLifecycleMismatch ReleaseErrorCode = "LIFECYCLE_MISMATCH"

	// ErrCodeValidationFailed means schema validation (lint/breaking/policy)
	// did not pass.
	ErrCodeValidationFailed ReleaseErrorCode = "VALIDATION_FAILED"

	// ErrCodeGoPackageMismatch means a proto file's go_package option does
	// not match the derived import path.
	ErrCodeGoPackageMismatch ReleaseErrorCode = "GO_PACKAGE_MISMATCH"

	// ErrCodeGoModMismatch means an existing go.mod has a different module
	// path than expected.
	ErrCodeGoModMismatch ReleaseErrorCode = "GO_MOD_MISMATCH"

	// ErrCodeMergeConflict means the canonical repo has diverged and a
	// merge conflict occurred during the release.
	ErrCodeMergeConflict ReleaseErrorCode = "MERGE_CONFLICT"

	// ErrCodeCanonicalMoved means the canonical repo's HEAD has moved since
	// the prepare step.
	ErrCodeCanonicalMoved ReleaseErrorCode = "CANONICAL_MOVED"

	// ErrCodePolicyFailed means a canonical CI policy check failed.
	ErrCodePolicyFailed ReleaseErrorCode = "POLICY_FAILED"

	// ErrCodePackageReleaseFailed means language package publication failed
	// after the canonical tag was created.
	ErrCodePackageReleaseFailed ReleaseErrorCode = "PACKAGE_RELEASE_FAILED"

	// ErrCodeCatalogUpdateFailed means the catalog update failed after
	// canonical release.
	ErrCodeCatalogUpdateFailed ReleaseErrorCode = "CATALOG_UPDATE_FAILED"

	// ErrCodeNotGitRepo means the current directory is not a git repository.
	ErrCodeNotGitRepo ReleaseErrorCode = "NOT_GIT_REPO"

	// ErrCodeInvalidVersion means the version string is not valid semver.
	ErrCodeInvalidVersion ReleaseErrorCode = "INVALID_VERSION"

	// ErrCodeMissingConfig means apx.yaml could not be loaded.
	ErrCodeMissingConfig ReleaseErrorCode = "MISSING_CONFIG"

	// ErrCodePushFailed means the git push to canonical repo failed.
	ErrCodePushFailed ReleaseErrorCode = "PUSH_FAILED"

	// ErrCodeBreakingChange means breaking changes were detected on the
	// current API line. A new major line is required.
	ErrCodeBreakingChange ReleaseErrorCode = "BREAKING_CHANGE"

	// ErrCodeVersionLineMismatch means the version's major component does
	// not match the API line's major version.
	ErrCodeVersionLineMismatch ReleaseErrorCode = "VERSION_LINE_MISMATCH"

	// ErrCodeIllegalTransition means a lifecycle transition is not allowed
	// (e.g. stable → experimental).
	ErrCodeIllegalTransition ReleaseErrorCode = "ILLEGAL_TRANSITION"

	// ErrCodePRCreationFailed means the pull request could not be created
	// on the canonical repo.
	ErrCodePRCreationFailed ReleaseErrorCode = "PR_CREATION_FAILED"
)

// ReleaseError is a structured error with a machine-readable code,
// human-readable message, and optional recovery hint.
type ReleaseError struct {
	Code    ReleaseErrorCode `json:"code"`
	Message string           `json:"message"`
	Hint    string           `json:"hint,omitempty"`
}

func (e *ReleaseError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("[%s] %s (hint: %s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewReleaseError creates a new ReleaseError.
func NewReleaseError(code ReleaseErrorCode, message string) *ReleaseError {
	return &ReleaseError{Code: code, Message: message}
}

// WithHint returns a copy of the error with a recovery hint attached.
func (e *ReleaseError) WithHint(hint string) *ReleaseError {
	return &ReleaseError{Code: e.Code, Message: e.Message, Hint: hint}
}

// ErrorCodeDescriptions maps each error code to a brief description
// suitable for documentation and CLI help.
var ErrorCodeDescriptions = map[ReleaseErrorCode]string{
	ErrCodeVersionTaken:         "Version already exists with different content",
	ErrCodeLifecycleBlocked:     "Lifecycle state blocks new releases",
	ErrCodeLifecycleMismatch:    "Version prerelease tag conflicts with lifecycle",
	ErrCodeValidationFailed:     "Schema validation did not pass",
	ErrCodeGoPackageMismatch:    "Proto go_package does not match derived import",
	ErrCodeGoModMismatch:        "go.mod module directive mismatch",
	ErrCodeMergeConflict:        "Merge conflict in canonical repo",
	ErrCodeCanonicalMoved:       "Canonical repo HEAD has moved since prepare",
	ErrCodePolicyFailed:         "Canonical CI policy check failed",
	ErrCodePackageReleaseFailed: "Package release failed after canonical tag",
	ErrCodeCatalogUpdateFailed:  "Catalog update failed after release",
	ErrCodeNotGitRepo:           "Not inside a git repository",
	ErrCodeInvalidVersion:       "Version is not valid semver",
	ErrCodeMissingConfig:        "apx.yaml not found or invalid",
	ErrCodePushFailed:           "Git push to canonical repo failed",
	ErrCodeBreakingChange:       "Breaking changes require a new API line",
	ErrCodeVersionLineMismatch:  "Version major does not match API line",
	ErrCodeIllegalTransition:    "Illegal lifecycle transition",
	ErrCodePRCreationFailed:     "PR creation on canonical repo failed",
}
