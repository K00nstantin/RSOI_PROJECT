package auth

import (
	"fmt"
	"io"
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

type authConfig struct {
	publicKey interface{}
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
