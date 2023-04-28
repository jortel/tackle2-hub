package api

//
// Routes
const (
	AuthRoot      = "/auth"
	AuthLoginRoot = AuthRoot + "/login"
)

//
// Login REST resource.
type Login struct {
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token"`
}
