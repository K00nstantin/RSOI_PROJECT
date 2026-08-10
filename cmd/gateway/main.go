package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type gatewayConfig struct {
	client            http.Client
	libraryServiceURL string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}
	libraryServiceURL := os.Getenv("LIBRARY_SERVICE_URL")

	myClient := http.Client{
		Timeout: 10 * time.Second,
	}

	cfg := gatewayConfig{
		client:            myClient,
		libraryServiceURL: libraryServiceURL,
	}

	r := gin.Default()

	r.GET("/api/v1/libraries", cfg.getLibrariesHandler)
	r.GET("/api/v1/libraries/:libraryUid/books", cfg.getLibraryBooksHandler)
	// r.GET("/api/v1/reservations", getReservationsHandler)
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
