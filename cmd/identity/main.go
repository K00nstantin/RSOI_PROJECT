package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type identityConfig struct {
	client             http.Client
	gatewayServiceURL  string
	identityServiceURL string
	jwkSet             []byte
	publicKey          *rsa.PublicKey
	privateKey         *rsa.PrivateKey
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

	if err := cfg.initKeys(); err != nil {
		fmt.Println("error initializing keys")
		return
	}

	r := gin.Default()

	r.GET("/api/v1/authorize", cfg.authorizationHandler)
	r.POST("/api/v1/callback", cfg.callbackHandler)
	r.GET("/api/v1/jwks", cfg.getJWKS)

	r.Run(":8090")
}

func (cfg *identityConfig) authorizationHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "this is authorization handler",
	})
}

func (cfg *identityConfig) callbackHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "this is callback handler",
	})
}

func (cfg *identityConfig) initKeys() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	publicKey := privateKey.PublicKey
	cfg.privateKey = privateKey
	cfg.publicKey = &publicKey
	key, err := jwk.FromRaw(publicKey)
	if err != nil {
		return err
	}
	_ = key.Set(jwk.KeyIDKey, "idp-key-1")
	_ = key.Set(jwk.AlgorithmKey, "RS256")
	_ = key.Set(jwk.KeyUsageKey, "sig")
	set := jwk.NewSet()
	set.AddKey(key)
	jwksJSON, err := json.Marshal(set)
	if err != nil {
		return err
	}
	cfg.jwkSet = jwksJSON
	return nil
}

func (cfg *identityConfig) getJWKS(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", cfg.jwkSet)
}
