package main

import (
	"RSOI_PROJECT/internal/auth"
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
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("error while loading .env")
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

	auth_cfg := auth.NewConfig()
	if err = auth_cfg.LoadJWKS(cfg.identityServiceURL); err != nil {
		log.Fatalf("Failed to load JWKS: %v", err)
	}

	server := gin.Default()
	server.Use(auth_cfg.AuthMiddleware())
	server.GET("/api/v1/rating", cfg.getRating)
	server.PUT("/api/v1/rating", cfg.updateRating)
	server.POST("/api/v1/rating/init", cfg.initRating)
	server.GET("/manage/health", healthCheck)

	log.Println("Rating service starting on :8050")
	if err := server.Run(":8050"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}

func (cfg *ratingConfig) getRating(c *gin.Context) {
	username := c.GetString("username")
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

func (cfg *ratingConfig) initRating(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Stars    int32  `json:"stars"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	params := ratingdb.CreateUserParams{
		Username: req.Username,
		Stars:    req.Stars,
	}
	if err := cfg.queries.CreateUser(c, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rating"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "rating initialized"})
}
