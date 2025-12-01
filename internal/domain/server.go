package domain

import "time"

type Server struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Loader    string    `json:"loader"`
	Port      int       `json:"port"`
	RAM       int       `json:"ram"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
