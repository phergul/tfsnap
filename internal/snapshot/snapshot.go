package snapshot

import "time"

type Provider struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	SchemaFile string `json:"schema_file"`
	Binary     string `json:"binary"`
}

type Metadata struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Provider  Provider  `json:"provider"`
}
