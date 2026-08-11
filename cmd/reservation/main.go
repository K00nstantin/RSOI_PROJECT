package main

import (
	"RSOI_PROJECT/internal/reservationdb"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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
	// server.GET("/api/v1/reservations/active/count", getActiveReservationsCount)
	// server.POST("/api/v1/reservations", createReservations)
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
