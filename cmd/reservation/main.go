package main

import (
	"RSOI_PROJECT/internal/reservationdb"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type reservationConfig struct {
	reservationServiceURL string
	queries               *reservationdb.Queries
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("error loading .env")
		return
	}
	db_url := os.Getenv("RESERVATIONS_DB_URL")
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Println("error opening database")
		return
	}
	dbQueries := reservationdb.New(db)
	reservationServiceURL := os.Getenv("RESERVATION_SERVICE_URL")
	cfg := reservationConfig{
		reservationServiceURL: reservationServiceURL,
		queries:               dbQueries,
	}
	server := gin.Default()
	server.GET("/api/v1/reservations", cfg.getReservations)
	server.GET("/api/v1/reservations/active/count", cfg.getActiveReservationsCount)
	server.POST("/api/v1/reservations", cfg.createReservation)
	// server.POST("/api/v1/reservations/:reservationUid/return", returnBook)
	// server.GET("/manage/health", healthCheck)

	log.Println("Reservation service starting on :8070")
	if err := server.Run(":8070"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (cfg *reservationConfig) getReservations(c *gin.Context) {
	username := c.Request.Header.Get("X-User-Name")
	reservations, err := cfg.queries.GetReservations(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make db query",
			"err":   err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reservations": reservations,
	})
}

func (cfg *reservationConfig) getActiveReservationsCount(c *gin.Context) {
	username := c.Request.Header.Get("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid username",
		})
		return
	}
	reservations, err := cfg.queries.GetActiveReservations(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make a query",
			"err":   err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(reservations),
	})
}

func (cfg *reservationConfig) createReservation(c *gin.Context) {
	username := c.Request.Header.Get("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid username",
		})
		return
	}
	type request_body struct {
		BookUid    uuid.UUID `json:"bookUid"`
		LibraryUid uuid.UUID `json:"libraryUid"`
		TillDate   time.Time `json:"tillDate"`
	}
	req_body := request_body{}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}
	if err = json.Unmarshal(body, &req_body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unmarshall",
			"err":   err,
		})
		return
	}
	params := reservationdb.CreateReservationParams{
		Username:   username,
		BookUid:    req_body.BookUid,
		LibraryUid: req_body.LibraryUid,
		TillDate:   req_body.TillDate,
	}
	row, err := cfg.queries.CreateReservation(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make query",
			"err":   err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"reservation": row,
	})
}
