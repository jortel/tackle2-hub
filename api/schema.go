package api

//
// Routes
const (
	SchemaRoot = "/schema"
)

type Schema struct {
	Version string   `json:"version,omitempty"`
	Paths   []string `json:"paths"`
}
