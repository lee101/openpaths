package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Disabled     bool      `json:"disabled"`
}

type APIKey struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	KeyHash      string     `json:"-"`
	KeyPrefix    string     `json:"key_prefix"`
	Name         string     `json:"name"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	Revoked      bool       `json:"revoked"`
	RateLimitRPM int        `json:"rate_limit_rpm"`
}
