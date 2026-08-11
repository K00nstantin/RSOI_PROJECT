package main

import (
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
)

type gatewayConfig struct {
	client                http.Client
	libraryServiceURL     string
	reservationServiceURL string
}

type book struct {
	BookUid uuid.UUID      `json:"book_uid"`
	Name    string         `json:"name"`
	Author  sql.NullString `json:"author"`
	Genre   sql.NullString `json:"genre"`
}

type library struct {
	LibraryUid uuid.UUID `json:"library_uid"`
	Name       string    `json:"name"`
	City       string    `json:"city"`
	Address    string    `json:"address"`
}

type bookDTO struct {
	BookUid uuid.UUID `json:"book_uid"`
	Name    string    `json:"name"`
	Author  string    `json:"author"`
	Genre   string    `json:"genre"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	libraryServiceURL := os.Getenv("LIBRARY_SERVICE_URL")
	reservationServiceURL := os.Getenv("RESERVATION_SERVICE_URL")

	myClient := http.Client{
		Timeout: 10 * time.Second,
	}

	cfg := gatewayConfig{
		client:                myClient,
		libraryServiceURL:     libraryServiceURL,
		reservationServiceURL: reservationServiceURL,
	}

	r := gin.Default()

	r.GET("/api/v1/libraries", cfg.getLibrariesHandler)
	r.GET("/api/v1/libraries/:libraryUid/books", cfg.getLibraryBooksHandler)
	r.GET("/api/v1/reservations", cfg.getReservationsHandler)
	// r.POST("/api/v1/reservations", createReservationHandler)
	// r.POST("/api/v1/reservations/:reservationUid/return", returnBookHandler)
	// r.GET("/api/v1/rating", getRatingHandler)
	r.GET("/manage/health", healthCheck)

	log.Println("Gateway service starting on port 8080")
	r.Run(":8080")
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}

func (cfg *gatewayConfig) getLibrariesHandler(c *gin.Context) {
	query_params := c.Request.URL.Query().Encode()

	request_string := cfg.libraryServiceURL + "/api/v1/libraries" + "?" + query_params

	req, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	resp, err := cfg.client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}

	c.Data(http.StatusOK, "application/json", body)
}

func (cfg *gatewayConfig) getLibraryBooksHandler(c *gin.Context) {
	libraryUid := c.Param("libraryUid")
	other_params := c.Request.URL.RawQuery
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + libraryUid + "/books?" + other_params

	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
		})
		return
	}

	resp, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to send request",
			"err":   err,
		})
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}

	c.Data(http.StatusOK, "application/json", body)
}

func (cfg *gatewayConfig) getReservationsHandler(c *gin.Context) {
	username := c.Request.Header.Get("X-User-Name")
	request_string := cfg.reservationServiceURL + "/api/v1/reservations"
	req, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
			"err":   err,
		})
		return
	}
	req.Header.Add("X-User-Name", username)
	resp, err := cfg.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "failed to perform request",
			"err":   err,
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error reading body",
			"err":   err,
		})
		return
	}
	type reservation struct {
		ID             int32     `json:"id"`
		ReservationUid uuid.UUID `json:"reservation_uid"`
		Username       string    `json:"username"`
		BookUid        uuid.UUID `json:"book_uid"`
		LibraryUid     uuid.UUID `json:"library_uid"`
		Status         string    `json:"status"`
		StartDate      time.Time `json:"start_date"`
		TillDate       time.Time `json:"till_date"`
	}

	type wrapper struct {
		Reservations []reservation `json:"reservations"`
	}
	wrapped := wrapper{}

	err = json.Unmarshal(body, &wrapped)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error decoding",
			"err":   err,
		})
		return
	}

	type final struct {
		ReservationUid uuid.UUID `json:"reservationUid"`
		Status         string    `json:"status"`
		StartDate      time.Time `json:"startDate"`
		TillDate       time.Time `json:"tillDate"`
		Book           bookDTO   `json:"book"`
		Library        library   `json:"library"`
	}

	final_resp := []final{}
	for _, reserv := range wrapped.Reservations {
		lib_response, err := cfg.getLibraryByUUID(c, reserv.LibraryUid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}

		lib_body, err := io.ReadAll(lib_response.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}
		lib := library{}
		if err = json.Unmarshal(lib_body, &lib); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}

		bk_response, err := cfg.getBookByUUID(c, reserv.BookUid, reserv.LibraryUid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}
		bk_body, err := io.ReadAll(bk_response.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}

		bk := book{}

		if err = json.Unmarshal(bk_body, &bk); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}
		dto := bookDTO{
			BookUid: bk.BookUid,
			Name:    bk.Name,
			Author:  bk.Author.String,
			Genre:   bk.Genre.String,
		}
		final_resp = append(final_resp, final{
			ReservationUid: reserv.ReservationUid,
			Status:         reserv.Status,
			StartDate:      reserv.StartDate,
			TillDate:       reserv.TillDate,
			Library:        lib,
			Book:           dto,
		})
	}

	c.JSON(http.StatusOK, final_resp)
}

func (cfg *gatewayConfig) getLibraryByUUID(c *gin.Context, uuid uuid.UUID) (http.Response, error) {
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + uuid.String()
	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while creating getlibrarybyuuid request",
			"err":   err,
		})
	}
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while making getlibrarybyuuid request",
			"err":   err,
		})
		return http.Response{}, err
	}
	return *response, nil
}

func (cfg *gatewayConfig) getBookByUUID(c *gin.Context, book_uuid uuid.UUID, libraryuuid uuid.UUID) (http.Response, error) {
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + libraryuuid.String() + "/books/" + book_uuid.String()
	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while creating getlibrarybyuuid request",
			"err":   err,
		})
		return http.Response{}, err
	}
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while making getbookbyuuid request",
			"err":   err,
		})
		return http.Response{}, err
	}
	return *response, nil
}
