package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/supabase-go"
)

// Get user patterns (cached insights)
func GetUserPatterns(client *supabase.Client, userID string) (types.UserPatterns, error) {
	resp, _, err := client.From("user_patterns").
		Select("*", "", false).
		Eq("user_id", userID).
		Execute()

	if err != nil {
		return types.UserPatterns{}, fmt.Errorf("failed to fetch user patterns: %w", err)
	}

	var patterns []types.UserPatterns
	if err := json.Unmarshal(resp, &patterns); err != nil {
		return types.UserPatterns{}, fmt.Errorf("failed to unmarshal patterns: %w", err)
	}

	// Return existing patterns or empty struct
	if len(patterns) > 0 {
		return patterns[0], nil
	}

	return types.UserPatterns{UserID: userID}, nil
}

// Update user patterns
func UpdateUserPatterns(client *supabase.Client, userID string, patterns types.UserPatterns) error {
	patterns.UserID = userID
	patterns.UpdatedAt = time.Now()

	// Upsert patterns
	_, _, err := client.From("user_patterns").
		Upsert(patterns, "", "", "user_id").
		Execute()

	if err != nil {
		return fmt.Errorf("failed to update user patterns: %w", err)
	}

	return nil
}
