package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"RSOI_PROJECT/pkg/models" // или ваш путь к модулю
)

var db *gorm.DB
var privateKey *rsa.PrivateKey
var publicKey *rsa.PublicKey

// Хранилище кодов авторизации (в памяти)
var authCodes = struct {
	sync.RWMutex
	m map[string]codeData
}{m: make(map[string]codeData)}

type codeData struct {
	Username  string
	ExpiresAt time.Time
}

func main() {
	log.Println("Starting identity service...")

	// Подключение к БД (используем тот же PostgreSQL, но базу данных identity)
	host := getEnv("DB_HOST", "postgres")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "program")
	password := getEnv("DB_PASSWORD", "test")
	dbname := getEnv("DB_NAME", "identity") // отдельная БД для identity

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Database connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Миграция
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Database connected and migrated")

	// Генерация или загрузка RSA ключей
	if err := loadOrGenerateKeys(); err != nil {
		log.Fatalf("Keys error: %v", err)
	}

	// Создание администратора (если не существует)
	createAdminUser()

	// Настройка роутера
	r := gin.Default()

	// Публичные эндпоинты
	r.POST("/auth/login", loginHandler)
	r.POST("/auth/token", tokenHandler)
	r.GET("/auth/jwks", jwksHandler)

	// Защищённые эндпоинты (только для Admin)
	adminGroup := r.Group("/api/v1")
	adminGroup.Use(AdminMiddleware())
	{
		adminGroup.POST("/users", createUserHandler)
	}

	// Healthcheck
	r.GET("/manage/health", healthCheck)

	log.Println("Identity service starting on :8040")
	if err := r.Run(":8040"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// ------------------ Обработчики ------------------

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginHandler – выдача одноразового кода (Authorization Code)
func loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user models.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Генерируем код
	code := uuid.New().String()
	authCodes.Lock()
	authCodes.m[code] = codeData{
		Username:  user.Username,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	authCodes.Unlock()

	// Возвращаем код (в реальном OIDC здесь должен быть redirect_uri, но мы упростим)
	c.JSON(http.StatusOK, gin.H{"code": code})
}

type tokenRequest struct {
	GrantType string `json:"grant_type" form:"grant_type"`
	Code      string `json:"code" form:"code"`
	ClientID  string `json:"client_id" form:"client_id"` // не обязателен
}

// tokenHandler – обмен кода на JWT (id_token + access_token)
func tokenHandler(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBind(&req); err != nil || req.GrantType != "authorization_code" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	authCodes.RLock()
	data, exists := authCodes.m[req.Code]
	authCodes.RUnlock()

	if !exists || time.Now().After(data.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired code"})
		return
	}

	// Удаляем код (одноразовый)
	authCodes.Lock()
	delete(authCodes.m, req.Code)
	authCodes.Unlock()

	// Получаем пользователя
	var user models.User
	if err := db.Where("username = ?", data.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	// Генерируем id_token (JWT)
	idToken, err := generateJWT(user.Username, user.Role, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	// Для простоты возвращаем тот же токен как access_token
	c.JSON(http.StatusOK, gin.H{
		"access_token": idToken,
		"id_token":     idToken,
		"token_type":   "Bearer",
		"expires_in":   900,
	})
}

// jwksHandler – выдаёт JWKS (публичный ключ)
func jwksHandler(c *gin.Context) {
	if publicKey == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no public key"})
		return
	}
	// Экспорт публичного ключа в формат JWK
	n := publicKey.N.Bytes()
	e := []byte{0x01, 0x00, 0x01} // 65537

	jwk := map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": "primary",
		"n":   base64URLEncode(n),
		"e":   base64URLEncode(e),
	}

	c.JSON(http.StatusOK, gin.H{
		"keys": []map[string]interface{}{jwk},
	})
}

// createUserHandler – создание нового пользователя (только для Admin)
type createUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email"`
	Role     string `json:"role"` // "admin" или "user"
}

func createUserHandler(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что роль допустима
	if req.Role != "" && req.Role != "admin" && req.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'admin' or 'user'"})
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}

	// Хешируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Email:        req.Email,
		Role:         req.Role,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"username": user.Username,
		"role":     user.Role,
		"email":    user.Email,
	})
}

// healthCheck
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"details": "Identity service is active",
	})
}

// ------------------ Middleware для проверки роли Admin ------------------

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		// Валидируем токен (используем публичный ключ)
		token, err := validateJWT(tokenString)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		role, ok := claims["role"].(string)
		if !ok || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Next()
	}
}

// ------------------ JWT и ключи ------------------

func loadOrGenerateKeys() error {
	// Попытка загрузить ключи из файлов
	privPEM, err := os.ReadFile("keys/private.pem")
	if err == nil {
		block, _ := pem.Decode(privPEM)
		if block != nil {
			privKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err == nil {
				var ok bool
				privateKey, ok = privKeyInterface.(*rsa.PrivateKey)
				if ok {
					pubPEM, err := os.ReadFile("keys/public.pem")
					if err == nil {
						block, _ := pem.Decode(pubPEM)
						if block != nil {
							pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
							if err == nil {
								publicKey, ok = pubKeyInterface.(*rsa.PublicKey)
								if ok {
									log.Println("Loaded existing RSA keys")
									return nil
								}
							}
						}
					}
				}
			}
		}
	}

	// Если не загрузились – генерируем новые
	log.Println("Generating new RSA keys...")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	privateKey = priv
	publicKey = &priv.PublicKey

	// Сохраняем в файлы (опционально)
	os.MkdirAll("keys", 0700)
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEMBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}
	if err := os.WriteFile("keys/private.pem", pem.EncodeToMemory(privPEMBlock), 0600); err != nil {
		log.Printf("Warning: failed to save private key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	pubPEMBlock := &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}
	if err := os.WriteFile("keys/public.pem", pem.EncodeToMemory(pubPEMBlock), 0644); err != nil {
		log.Printf("Warning: failed to save public key: %v", err)
	}
	return nil
}

func generateJWT(username, role string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  username,
		"role": role,
		"iat":  now.Unix(),
		"exp":  now.Add(expiry).Unix(),
		"jti":  uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return publicKey, nil
	})
}

// Извлечение токена из заголовка Authorization
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// base64URLEncode для JWK
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// ------------------ Вспомогательные ------------------

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func createAdminUser() {
	var count int64
	db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		// Создаём администратора по умолчанию
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{
			Username:     "admin",
			PasswordHash: string(hash),
			Email:        "admin@example.com",
			Role:         "admin",
		}
		if err := db.Create(&admin).Error; err != nil {
			log.Printf("Failed to create admin user: %v", err)
		} else {
			log.Println("Created default admin user: admin/admin123")
		}
	}
}
