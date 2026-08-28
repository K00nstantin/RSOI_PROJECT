package main

import (
	"RSOI_PROJECT/internal/auth"
	"RSOI_PROJECT/internal/identitydb"
	"RSOI_PROJECT/models"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v2/jwk"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type identityConfig struct {
	client             http.Client
	gatewayServiceURL  string
	identityServiceURL string
	jwkSet             []byte
	publicKey          *rsa.PublicKey
	privateKey         *rsa.PrivateKey
	queries            *identitydb.Queries
}

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println("error loading .env")
	}
	db_url := os.Getenv("IDENTITY_DB_URL")
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Println("error openning db")
	}
	dbQueries := identitydb.New(db)

	myClient := http.Client{
		Timeout: 10 * time.Second,
	}

	cfg := identityConfig{
		client:             myClient,
		gatewayServiceURL:  os.Getenv("GATEWAY_SERVICE_URL"),
		identityServiceURL: os.Getenv("IDENTITY_SERVICE_URL"),
		queries:            dbQueries,
	}
	auth_cfg := auth.NewConfig()
	if err := cfg.initKeys(); err != nil {
		fmt.Println("error initializing keys")
		return
	}
	auth_cfg.SetPublicKey(cfg.publicKey)

	adminScopes := []string{"openid", "profile", "email"}
	adminJWT, err := cfg.generateJWT("admin", "admin_email", "Admin", adminScopes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(adminJWT)

	r := gin.Default()
	r.Use(auth_cfg.AuthMiddleware())
	r.GET("/api/v1/authorize", cfg.authorizationHandler)
	r.GET("/api/v1/jwks", cfg.getJWKS)
	r.POST("/api/v1/createUser", cfg.createUserHandler)

	r.Run(":8090")
}

func (cfg *identityConfig) authorizationHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "this is authorization handler",
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

func (cfg *identityConfig) generateJWT(username, email, role string, scopes []string) (string, error) {
	now := time.Now()
	claims := auth.Claims{
		Sub:  username,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour * 1)),
		},
	}
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[scope] = true
	}
	if scopeMap["email"] {
		claims.Email = email
	}
	if scopeMap["profile"] {
		claims.Name = username
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "idp-key-1"
	return token.SignedString(cfg.privateKey)
}

func (cfg *identityConfig) getUserByUsername(c *gin.Context, username string) (models.User, error) {
	db_user, err := cfg.queries.GetUserByUsername(c, username)
	if err != nil {
		return models.User{}, err
	}
	user := models.User{
		Username:     db_user.Username,
		PasswordHash: db_user.PasswordHash,
		Email:        db_user.Email.String,
		Role:         db_user.Role,
	}
	return user, nil
}

func (cfg *identityConfig) createUserHandler(c *gin.Context) {
	if c.GetString("role") != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "role must be admin",
		})
		return
	}
	var unhashed_user struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&unhashed_user); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}
	hashed_password, err := bcrypt.GenerateFromPassword([]byte(unhashed_user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while hashing password",
		})
		return
	}
	email := sql.NullString{
		String: unhashed_user.Email,
		Valid:  true,
	}
	params := identitydb.InsertUserParams{
		Username:     unhashed_user.Username,
		PasswordHash: string(hashed_password),
		Email:        email,
		Role:         unhashed_user.Role,
	}
	if err = cfg.queries.InsertUser(c, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while making db query",
		})
		return
	}
	c.JSON(http.StatusCreated, nil)
}
