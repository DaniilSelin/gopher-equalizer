package models

import "time"

type Bucket struct {
	ClientID   string    `json:"client_id"`
	Capacity   int       `json:"capacity"`
	Tokens     int       `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
}
