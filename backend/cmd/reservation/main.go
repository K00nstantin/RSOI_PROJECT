package main

import (
	"RSOI_PROJECT/internal/auth"
	"RSOI_PROJECT/internal/models"
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
	identityServiceURL    string
	client                http.Client
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
	myClient := http.Client{
		Timeout: 10 * time.Second,
	}
	cfg := reservationConfig{
		reservationServiceURL: os.Getenv("RESERVATION_SERVICE_URL"),
		queries:               dbQueries,
		client:                myClient,
		identityServiceURL:    os.Getenv("IDENTITY_SERVICE_URL"),
	}

	auth_cfg := auth.NewConfig()
	if err = auth_cfg.LoadJWKS(cfg.identityServiceURL); err != nil {
		log.Fatalf("Failed to load JWKS: %v", err)
	}

	server := gin.Default()
	server.Use(auth_cfg.AuthMiddleware())
	server.GET("/api/v1/reservations", cfg.getReservations)
	server.GET("/api/v1/reservations/active/count", cfg.getActiveReservationsCount)
	server.POST("/api/v1/reservations", cfg.createReservation)
	server.POST("/api/v1/reservations/:reservationUid/return", cfg.returnBook)
	server.GET("/manage/health", healthCheck)

	log.Println("Reservation service starting on :8070")
	if err := server.Run(":8070"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (cfg *reservationConfig) getReservations(c *gin.Context) {
	username := c.GetString("username")
	reservations, err := cfg.queries.GetReservations(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make db query",
			"err":   err,
		})
		return
	}
	final := make([]models.Reservation, len(reservations))
	for i, res := range reservations {
		final[i].ID = res.ID
		final[i].ReservationUid = res.ReservationUid
		final[i].Username = res.Username
		final[i].BookUid = res.BookUid
		final[i].LibraryUid = res.LibraryUid
		final[i].Status = res.Status
		final[i].StartDate = models.Date(res.StartDate)
		final[i].TillDate = models.Date(res.TillDate)

	}

	c.JSON(http.StatusOK, gin.H{
		"reservations": final,
	})
}

func (cfg *reservationConfig) getActiveReservationsCount(c *gin.Context) {
	username := c.GetString("username")
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
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid username",
		})
		return
	}

	req_body := models.CreateReservationBody{}
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
		TillDate:   time.Time(req_body.TillDate),
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

func (cfg *reservationConfig) returnBook(c *gin.Context) {
	delta := 0
	type sqlrow struct {
		BookUid    uuid.UUID `json:"book_uid"`
		LibraryUid uuid.UUID `json:"library_uid"`
	}
	var row sqlrow
	flag := false
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error reading body",
			"err":   err,
		})
		return
	}
	params := models.CloseReservationParams{}
	err = json.Unmarshal(body, &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while unmarshalling",
			"err":   err,
		})
		return
	}
	reservationUid := c.Param("reservationUid")
	id, err := uuid.Parse(reservationUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid id",
			"err":   err,
		})
		return
	}
	reservation, err := cfg.queries.GetReservation(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid parameters",
			"err":   err,
		})
		return
	}

	if time.Time(params.Date).After(reservation.TillDate) {
		delta -= 10
		flag = true
		r, err := cfg.queries.SetExpired(c, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid parameters",
				"err":   err,
			})
			return
		}
		row.BookUid = r.BookUid
		row.LibraryUid = r.LibraryUid

	}
	if flag == false {
		delta++
		r, err := cfg.queries.SetReturned(c, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid parameters",
				"err":   err,
			})
			return
		}
		row.BookUid = r.BookUid
		row.LibraryUid = r.LibraryUid
	}
	c.JSON(http.StatusOK, gin.H{
		"bookUid":    row.BookUid,
		"libraryUid": row.LibraryUid,
		"delta":      delta,
	})

}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}
