package supabase

import (
	"clementus360/ai-helper/config"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/supabase-community/supabase-go"
)

var Client *supabase.Client

func Init() {
	apiURL := os.Getenv("SUPABASE_URL")
	apiKey := os.Getenv("SUPABASE_KEY")

	if apiURL == "" || apiKey == "" {
		config.Logger.Fatal("SUPABASE_URL or SUPABASE_KEY is missing")
	}

	var err error
	Client, err = supabase.NewClient(apiURL, apiKey, &supabase.ClientOptions{})
	if err != nil {
		config.Logger.Fatal("Failed to create Supabase client:", err)
	}
}

func SupabaseClientFromRequest(r *http.Request) (*supabase.Client, string, error) {
	apiURL := os.Getenv("SUPABASE_URL")
	apiKey := os.Getenv("SUPABASE_KEY")

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, "", fmt.Errorf("missing Authorization header")
	}

	jwtString := strings.TrimPrefix(authHeader, "Bearer ")
	if jwtString == "" {
		return nil, "", fmt.Errorf("invalid Authorization header")
	}

	// Parse the JWT
	token, _, err := new(jwt.Parser).ParseUnverified(jwtString, jwt.MapClaims{})
	if err != nil {
		return nil, "", fmt.Errorf("invalid JWT format")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, "", fmt.Errorf("invalid JWT claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, "", fmt.Errorf("missing sub in token")
	}

	// fmt.Println(GenerateTestJWT(userId))

	client, err := supabase.NewClient(apiURL, apiKey, &supabase.ClientOptions{
		Headers: map[string]string{
			"Authorization": "Bearer " + jwtString,
		},
	})
	return client, sub, err
}
