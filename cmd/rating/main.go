package main

import (
	"RSOI_PROJECT/internal/ratingdb"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ratingConfig struct {
	ratingServiceURL   string
	queries            *ratingdb.Queries
	identityServiceURL string
	client             http.Client
	jwkSet             []byte
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("error while loading .env")
		return
	}

	db_url := os.Getenv("RATINGS_DB_URL")
	database, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Println("error opening database")
		return
	}

	queries := ratingdb.New(database)
	myClient := http.Client{
		Timeout: 10 * time.Second,
	}
	cfg := ratingConfig{
		ratingServiceURL:   db_url,
		queries:            queries,
		identityServiceURL: os.Getenv("IDENTITY_SERVICE_URL"),
		client:             myClient,
	}

	if err = cfg.getJWKS(); err != nil {
		fmt.Println("Error getting JWKS: %w", err)
	}

	server := gin.Default()
	server.GET("/api/v1/rating", cfg.getRating)
	server.PUT("/api/v1/rating", cfg.updateRating)
	// server.GET("/manage/health", healthCheck)

	log.Println("Rating service starting on :8050")
	if err := server.Run(":8050"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (cfg *ratingConfig) getRating(c *gin.Context) {
	username := c.Request.Header.Get("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid username",
		})
		return
	}
	stars, err := cfg.queries.GetUserStars(c, username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"stars": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stars": int(stars),
	})
}

func (cfg *ratingConfig) updateRating(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}
	params := ratingdb.UpdateRatingParams{}
	if err = json.Unmarshal(body, &params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unmarshall",
			"err":   err,
		})
		return
	}
	stars, err := cfg.queries.GetUserStars(c, params.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get rating",
			"err":   err,
		})
		return
	}
	if stars+params.Stars > 100 {
		params.Stars = 100 - stars
	}

	if err = cfg.queries.UpdateRating(c, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update rating",
			"err":   err,
		})
		return
	}

}

func (cfg *ratingConfig) getJWKS() error {
	request_str := cfg.identityServiceURL + "/api/v1/jwks"
	request, err := http.NewRequest("GET", request_str, nil)
	if err != nil {
		return fmt.Errorf("error while creating a request: %w", err)
	}
	response, err := cfg.client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		return fmt.Errorf("error while making a request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error while reading body: %w", err)
	}
	cfg.jwkSet = body
	return nil

}
