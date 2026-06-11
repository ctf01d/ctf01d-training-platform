package games

import "time"

type GameStatus string

const (
	StatusUpcoming GameStatus = "upcoming"
	StatusOngoing  GameStatus = "ongoing"
	StatusPast     GameStatus = "past"
	StatusUnknown  GameStatus = "unknown"
)

type RegistrationStatus string

const (
	RegUnscheduled RegistrationStatus = "unscheduled"
	RegUpcoming    RegistrationStatus = "upcoming"
	RegOpen        RegistrationStatus = "open"
	RegClosed      RegistrationStatus = "closed"
)

type ScoreboardStatus string

const (
	ScoreAlways   ScoreboardStatus = "always"
	ScoreUpcoming ScoreboardStatus = "upcoming"
	ScoreOpen     ScoreboardStatus = "open"
	ScoreClosed   ScoreboardStatus = "closed"
)

func ComputeStatus(startsAt, endsAt *time.Time, now time.Time) GameStatus {
	if startsAt != nil && endsAt != nil {
		if !startsAt.After(now) && !endsAt.Before(now) {
			return StatusOngoing
		}
	}
	if startsAt != nil && startsAt.After(now) {
		return StatusUpcoming
	}
	if endsAt != nil && endsAt.Before(now) {
		return StatusPast
	}
	return StatusUnknown
}

func ComputeRegistrationStatus(opensAt, closesAt *time.Time, now time.Time) RegistrationStatus {
	if opensAt == nil && closesAt == nil {
		return RegUnscheduled
	}
	if opensAt != nil && now.Before(*opensAt) {
		return RegUpcoming
	}
	afterOpen := opensAt == nil || !now.Before(*opensAt)
	beforeClose := closesAt == nil || !now.After(*closesAt)
	if afterOpen && beforeClose {
		return RegOpen
	}
	return RegClosed
}

func ComputeScoreboardStatus(opensAt, closesAt *time.Time, now time.Time) ScoreboardStatus {
	if opensAt == nil && closesAt == nil {
		return ScoreAlways
	}
	if opensAt != nil && now.Before(*opensAt) {
		return ScoreUpcoming
	}
	afterOpen := opensAt == nil || !now.Before(*opensAt)
	beforeClose := closesAt == nil || !now.After(*closesAt)
	if afterOpen && beforeClose {
		return ScoreOpen
	}
	return ScoreClosed
}
