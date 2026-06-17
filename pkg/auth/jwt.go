package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwksURL      string
	jwksCache    map[string]*rsa.PublicKey
	cacheMutex   sync.RWMutex
	lastUpdate   time.Time
	cacheTimeout = 1 * time.Hour
)

// InitJWKS инициализирует загрузку JWKS из identity сервиса
func InitJWKS(url string) {
	jwksURL = url
	jwksCache = make(map[string]*rsa.PublicKey)
	refreshJWKS()
}

// JWTMiddleware – Gin middleware для проверки JWT
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token, err := validateToken(tokenString)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		username, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no subject"})
			return
		}
		role, _ := claims["role"].(string)
		c.Set("username", username)
		c.Set("role", role)
		c.Next()
	}
}

// extractToken из заголовка Authorization
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

// validateToken проверяет JWT, используя JWKS
func validateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found")
		}
		key, err := getPublicKey(kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	})
	return token, err
}

// getPublicKey загружает ключ из кэша или обновляет JWKS
func getPublicKey(kid string) (*rsa.PublicKey, error) {
	cacheMutex.RLock()
	key, ok := jwksCache[kid]
	cacheMutex.RUnlock()
	if ok {
		return key, nil
	}
	// Обновляем кэш
	if err := refreshJWKS(); err != nil {
		return nil, err
	}
	cacheMutex.RLock()
	key, ok = jwksCache[kid]
	cacheMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("kid %s not found", kid)
	}
	return key, nil
}

// refreshJWKS загружает JWKS из identity сервиса
func refreshJWKS() error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if time.Since(lastUpdate) < cacheTimeout && len(jwksCache) > 0 {
		return nil
	}

	resp, err := http.Get(jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks request failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}
	newCache := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}
		pubKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}
		newCache[key.Kid] = pubKey
	}
	jwksCache = newCache
	lastUpdate = time.Now()
	return nil
}
