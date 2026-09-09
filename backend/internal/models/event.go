package models

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

func NewEvent(userID, action, details string) Event {
	return Event{
		EventID:   uuid.New().String(),
		UserID:    userID,
		Action:    action,
		Details:   details,
		Timestamp: time.Now(),
	}
}
