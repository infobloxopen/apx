package publisher

import "fmt"

// ReleaseState represents a stage in the release state machine. A release
// progresses through these states linearly, with any state able to
// transition to StateFailed.
type ReleaseState string

const (
	// StateDraft is the initial state when a release is first created.
	StateDraft ReleaseState = "draft"

	// StateValidated means schema validation (lint, breaking, policy) passed.
	StateValidated ReleaseState = "validated"

	// StateVersionSelected means the version has been computed and confirmed.
	StateVersionSelected ReleaseState = "version-selected"

	// StatePrepared means the release manifest is complete and ready to submit.
	// This is the output of `apx release prepare`.
	StatePrepared ReleaseState = "prepared"

	// StateSubmitted means the release request has been sent (PR created)
	// to the canonical repo.
	StateSubmitted ReleaseState = "submitted"

	// StateCanonicalPROpen means a PR is open against the canonical repo.
	StateCanonicalPROpen ReleaseState = "canonical-pr-open"

	// StateCanonicalValidated means canonical CI has re-validated the release.
	StateCanonicalValidated ReleaseState = "canonical-validated"

	// StateCanonicalReleased means the canonical tag has been created.
	StateCanonicalReleased ReleaseState = "canonical-released"

	// StatePackagePublished means language packages have been published.
	StatePackagePublished ReleaseState = "package-published"

	// StateFailed means the release encountered a terminal error.
	// The manifest's Error field has details.
	StateFailed ReleaseState = "failed"
)

// allStates lists states in progression order (excluding failed).
var allStates = []ReleaseState{
	StateDraft,
	StateValidated,
	StateVersionSelected,
	StatePrepared,
	StateSubmitted,
	StateCanonicalPROpen,
	StateCanonicalValidated,
	StateCanonicalReleased,
	StatePackagePublished,
}

// stateIndex maps each state to its ordinal position.
var stateIndex map[ReleaseState]int

func init() {
	stateIndex = make(map[ReleaseState]int, len(allStates))
	for i, s := range allStates {
		stateIndex[s] = i
	}
}

// ValidateTransition checks whether moving from `current` to `next` is
// a legal state transition.
//
// Rules:
//  1. Any state can transition to StateFailed.
//  2. StateFailed cannot transition to anything (requires a new release).
//  3. Otherwise, next must be strictly later in the progression.
func ValidateTransition(current, next ReleaseState) error {
	// Rule 1: any → failed is always allowed.
	if next == StateFailed {
		return nil
	}

	// Rule 2: failed is terminal.
	if current == StateFailed {
		return fmt.Errorf("cannot transition from %q: release is in a terminal failed state; start a new release", current)
	}

	curIdx, curOk := stateIndex[current]
	nextIdx, nextOk := stateIndex[next]
	if !curOk {
		return fmt.Errorf("unknown release state %q", current)
	}
	if !nextOk {
		return fmt.Errorf("unknown release state %q", next)
	}

	// Rule 3: forward-only.
	if nextIdx <= curIdx {
		return fmt.Errorf("illegal transition: cannot go from %q to %q (must move forward)", current, next)
	}

	return nil
}

// IsTerminal reports whether the state is a final state (success or failure).
func IsTerminal(state ReleaseState) bool {
	return state == StatePackagePublished || state == StateFailed
}

// StateLabel returns a short human-readable label for a state.
func StateLabel(s ReleaseState) string {
	labels := map[ReleaseState]string{
		StateDraft:              "Draft",
		StateValidated:          "Validated",
		StateVersionSelected:    "Version Selected",
		StatePrepared:           "Prepared",
		StateSubmitted:          "Submitted",
		StateCanonicalPROpen:    "Canonical PR Open",
		StateCanonicalValidated: "Canonical Validated",
		StateCanonicalReleased:  "Canonical Released",
		StatePackagePublished:   "Package Published",
		StateFailed:             "Failed",
	}
	if label, ok := labels[s]; ok {
		return label
	}
	return string(s)
}
