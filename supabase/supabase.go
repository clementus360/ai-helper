package supabase

import (
	"clementus360/ai-helper/config"
	"fmt"
	"net/http"
	"os"
	"strings"

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

func SupabaseClientFromRequest(r *http.Request) (*supabase.Client, error) {
	apiURL := os.Getenv("SUPABASE_URL")
	apiKey := os.Getenv("SUPABASE_KEY")

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	jwt := strings.TrimPrefix(authHeader, "Bearer ")
	if jwt == "" {
		return nil, fmt.Errorf("invalid Authorization header")
	}

	client, err := supabase.NewClient(apiURL, apiKey, &supabase.ClientOptions{
		Headers: map[string]string{
			"Authorization": "Bearer " + jwt,
		},
	})
	return client, err
}
