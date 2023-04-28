package api

//
// Routes
const (
	CacheRoot    = "/cache"
	CacheDirRoot = CacheRoot + "/*" + Wildcard
)

//
// Cache REST resource.
type Cache struct {
	Path     string `json:"path"`
	Capacity string `json:"capacity"`
	Used     string `json:"used"`
	Exists   bool   `json:"exists"`
}
