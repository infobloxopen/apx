package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTransition_ForwardAllowed(t *testing.T) {
	tests := []struct {
		from, to ReleaseState
	}{
		{StateDraft, StateValidated},
		{StateDraft, StatePrepared},
		{StateValidated, StateVersionSelected},
		{StateVersionSelected, StatePrepared},
		{StatePrepared, StateSubmitted},
		{StateSubmitted, StateCanonicalPROpen},
		{StateCanonicalPROpen, StateCanonicalValidated},
		{StateCanonicalValidated, StateCanonicalReleased},
		{StateCanonicalReleased, StatePackagePublished},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			assert.NoError(t, ValidateTransition(tt.from, tt.to))
		})
	}
}

func TestValidateTransition_SkipAllowed(t *testing.T) {
	// Skipping intermediate states (e.g. draft → prepared) is allowed
	assert.NoError(t, ValidateTransition(StateDraft, StatePrepared))
	assert.NoError(t, ValidateTransition(StateValidated, StatePrepared))
	assert.NoError(t, ValidateTransition(StateDraft, StatePackagePublished))
}

func TestValidateTransition_AnyToFailed(t *testing.T) {
	for _, s := range allStates {
		assert.NoError(t, ValidateTransition(s, StateFailed),
			"any state should be able to transition to failed")
	}
}

func TestValidateTransition_FailedIsTerminal(t *testing.T) {
	for _, s := range allStates {
		err := ValidateTransition(StateFailed, s)
		assert.Error(t, err, "failed state should not transition to %s", s)
	}
}

func TestValidateTransition_BackwardBlocked(t *testing.T) {
	tests := []struct {
		from, to ReleaseState
	}{
		{StateValidated, StateDraft},
		{StatePrepared, StateValidated},
		{StateSubmitted, StatePrepared},
		{StatePackagePublished, StateDraft},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			assert.Error(t, ValidateTransition(tt.from, tt.to))
		})
	}
}

func TestValidateTransition_SameStateBlocked(t *testing.T) {
	for _, s := range allStates {
		err := ValidateTransition(s, s)
		assert.Error(t, err, "transitioning to same state %s should be blocked", s)
	}
}

func TestIsTerminal(t *testing.T) {
	assert.True(t, IsTerminal(StatePackagePublished))
	assert.True(t, IsTerminal(StateFailed))
	assert.False(t, IsTerminal(StateDraft))
	assert.False(t, IsTerminal(StatePrepared))
	assert.False(t, IsTerminal(StateSubmitted))
}

func TestStateLabel(t *testing.T) {
	assert.Equal(t, "Draft", StateLabel(StateDraft))
	assert.Equal(t, "Failed", StateLabel(StateFailed))
	assert.Equal(t, "Prepared", StateLabel(StatePrepared))
	assert.Equal(t, "Package Published", StateLabel(StatePackagePublished))
}
