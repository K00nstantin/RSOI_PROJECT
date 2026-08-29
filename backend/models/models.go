package models

import (
	"database/sql"
	"fmt"
	"strings"
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
	Date      Date
}
type Library struct {
	LibraryUid uuid.UUID `json:"libraryUid"`
	Name       string    `json:"name"`
	City       string    `json:"city"`
	Address    string    `json:"address"`
}

type Book struct {
	BookUid        uuid.UUID `json:"bookUid"`
	Name           string    `json:"name"`
	Author         string    `json:"author"`
	Genre          string    `json:"genre"`
	Condition      string    `json:"condition"`
	AvaliableCount int32     `json:"avaliableCount"`
}

type CreateReservationBody struct {
	BookUid    uuid.UUID `json:"bookUid"`
	LibraryUid uuid.UUID `json:"libraryUid"`
	TillDate   Date      `json:"tillDate"`
}

type Date time.Time

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	*d = Date(t)
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	t := time.Time(d)
	return []byte(fmt.Sprintf(`"%s"`, t.Format("2006-01-02"))), nil
}

type GetReservationResponse struct {
	ReservationUid uuid.UUID `json:"reservationUid"`
	Status         string    `json:"status"`
	StartDate      Date      `json:"startDate"`
	TillDate       Date      `json:"tillDate"`
	Book           BookDTO   `json:"book"`
	Library        Library   `json:"library"`
}

type ReservationDTO struct {
	ReservationUid uuid.UUID `json:"reservationUid"`
	Status         string    `json:"status"`
	StartDate      Date      `json:"startDate"`
	TillDate       Date      `json:"tillDate"`
	Book           BookDTO   `json:"book"`
	Library        Library   `json:"library"`
	Rating         Rating    `json:"rating"`
}

type Reservation struct {
	ID             int32     `json:"id"`
	ReservationUid uuid.UUID `json:"reservation_uid"`
	Username       string    `json:"username"`
	BookUid        uuid.UUID `json:"book_uid"`
	LibraryUid     uuid.UUID `json:"library_uid"`
	Status         string    `json:"status"`
	StartDate      Date      `json:"start_date"`
	TillDate       Date      `json:"till_date"`
}

type Rating struct {
	Stars int `json:"stars"`
}

type BookDTO struct {
	BookUid uuid.UUID `json:"bookUid"`
	Name    string    `json:"name"`
	Author  string    `json:"author"`
	Genre   string    `json:"genre"`
}

type SQLbook struct {
	BookUid uuid.UUID      `json:"book_uid"`
	Name    string         `json:"name"`
	Author  sql.NullString `json:"author"`
	Genre   sql.NullString `json:"genre"`
}

type User struct {
	Username     string
	PasswordHash string
	Email        string
	Role         string
}
