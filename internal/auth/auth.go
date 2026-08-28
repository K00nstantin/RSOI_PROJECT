package auth

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type authConfig struct {
	publicKey interface{}
}

type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Name  string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

func NewConfig() *authConfig {
	return &authConfig{}
}

func (cfg *authConfig) LoadJWKS(idpURL string) error {
	request_str := idpURL + "/api/v1/jwks"
	response, err := http.Get(request_str)
	if err != nil || response.StatusCode != http.StatusOK {
		return fmt.Errorf("error while making a request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error while reading body: %w", err)
	}
	set, err := jwk.Parse(body)
	if err != nil {
		return fmt.Errorf("failed to parse body: %w", err)
	}
	key, ok := set.Key(0)
	if !ok {
		return fmt.Errorf("failed to get key")
	}
	var raw_key interface{}
	if err := key.Raw(&raw_key); err != nil {
		return fmt.Errorf("failed to get raw key: %w", err)
	}
	cfg.publicKey = raw_key
	return nil
}

func (cfg *authConfig) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		open_paths := map[string]bool{
			"/manage/health":    true,
			"/api/v1/jwks":      true,
			"/api/v1/authorize": true,
			"/api/v1/login":     true,
		}
		if open_paths[c.Request.URL.Path] {
			c.Next()
			return
		}
		auth_header := c.GetHeader("Authorization")
		if auth_header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing token",
			})
			return
		}
		parts := strings.Split(auth_header, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}
		token_str := parts[1]
		claims, err := cfg.ValidateToken(token_str)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.Set("username", claims.Sub)
		c.Set("role", claims.Role)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Next()
	}
}

func (cfg *authConfig) ValidateToken(token_string string) (*Claims, error) {
	if cfg.publicKey == nil {
		return nil, fmt.Errorf("publicKey not loaded")
	}
	token, err := jwt.ParseWithClaims(token_string, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return cfg.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (cfg *authConfig) SetPublicKey(key interface{}) {
	cfg.publicKey = key
}
