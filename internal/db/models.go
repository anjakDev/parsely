package db

import "time"

// Vocabulary represents a vocabulary item stored in the database
type Vocabulary struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
}
