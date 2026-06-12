package games

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeStatus(t *testing.T) {
	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	tests := []struct {
		name     string
		startsAt *time.Time
		endsAt   *time.Time
		want     GameStatus
	}{
		{"ongoing", &past, &future, StatusOngoing},
		{"upcoming", &future, nil, StatusUpcoming},
		{"past", nil, &past, StatusPast},
		{"unknown both nil", nil, nil, StatusUnknown},
		{"ongoing exact start", &now, &future, StatusOngoing},
		{"ongoing exact end", &past, &now, StatusOngoing},
		{"ongoing exact both", &now, &now, StatusOngoing},
		{"past with start", &past, &past, StatusPast},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ComputeStatus(tt.startsAt, tt.endsAt, now))
		})
	}
}

func TestComputeRegistrationStatus(t *testing.T) {
	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	tests := []struct {
		name     string
		opensAt  *time.Time
		closesAt *time.Time
		want     RegistrationStatus
	}{
		{"unscheduled", nil, nil, RegUnscheduled},
		{"upcoming", &future, nil, RegUpcoming},
		{"open with both", &past, &future, RegOpen},
		{"open only open set", &past, nil, RegOpen},
		{"open only close set", nil, &future, RegOpen},
		{"closed", &past, &past, RegClosed},
		{"closed future open", &future, nil, RegUpcoming},
		{"closed past close", nil, &past, RegClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ComputeRegistrationStatus(tt.opensAt, tt.closesAt, now))
		})
	}
}

func TestComputeScoreboardStatus(t *testing.T) {
	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	tests := []struct {
		name     string
		opensAt  *time.Time
		closesAt *time.Time
		want     ScoreboardStatus
	}{
		{"always", nil, nil, ScoreAlways},
		{"upcoming", &future, nil, ScoreUpcoming},
		{"open with both", &past, &future, ScoreOpen},
		{"open only open set", &past, nil, ScoreOpen},
		{"open only close set", nil, &future, ScoreOpen},
		{"closed", &past, &past, ScoreClosed},
		{"closed future open", &future, nil, ScoreUpcoming},
		{"closed past close", nil, &past, ScoreClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ComputeScoreboardStatus(tt.opensAt, tt.closesAt, now))
		})
	}
}
