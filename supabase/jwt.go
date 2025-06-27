package supabase

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateTestJWT(userID string) (string, error) {
	secret := os.Getenv("SUPABASE_JWT_SECRET")

	claims := jwt.MapClaims{
		"sub":  userID,
		"aud":  "authenticated",
		"role": "authenticated",
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
