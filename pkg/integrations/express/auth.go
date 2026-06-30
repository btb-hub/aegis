package express

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func signBotID(botID, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(botID))
	return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
}

func verifyJWT(token, secretKey string) (map[string]any, error) {
	token = strings.TrimPrefix(strings.TrimSpace(token), "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}

	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid jwt signature encoding")
	}
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(signingInput))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("invalid jwt signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid jwt payload encoding")
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid jwt payload")
	}
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("jwt expired")
		}
	}
	return claims, nil
}

func VerifyAuthorization(authHeader, secretKey string) error {
	if strings.TrimSpace(authHeader) == "" {
		return fmt.Errorf("missing authorization header")
	}
	_, err := verifyJWT(authHeader, secretKey)
	return err
}
