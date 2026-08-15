package models

import (
	"time"

	"github.com/google/uuid"
)

type CloseReservationResponse struct {
	BookUid    uuid.UUID
	LibraryUid uuid.UUID
	Delta      int
}

type CloseReservationParams struct {
	Condition string
	Date      time.Time
}
