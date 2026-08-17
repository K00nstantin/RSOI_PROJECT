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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ratingConfig struct {
	ratingServiceURL string
	queries          *ratingdb.Queries
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
	cfg := ratingConfig{
		ratingServiceURL: db_url,
		queries:          queries,
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
