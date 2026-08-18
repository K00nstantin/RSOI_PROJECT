package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type identityConfig struct {
	client             http.Client
	gatewayServiceURL  string
	identityServiceURL string
}

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println("error loading .env")
	}

	myClient := http.Client{
		Timeout: 10 * time.Second,
	}

	cfg := identityConfig{
		client:             myClient,
		gatewayServiceURL:  os.Getenv("GATEWAY_SERVICE_URL"),
		identityServiceURL: os.Getenv("IDENTITY_SERVICE_URL"),
	}
	r := gin.Default()

	r.GET("/api/v1/authorize", cfg.authorizationHandler)
	r.POST("/api/v1/callback", cfg.callbackHandler)
}

func (cfg *identityConfig) authorizationHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "this is authorization handler",
	})
}

func (cfg *identityConfig) callbackHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "this is authorization handler",
	})
}
