package api

import "github.com/gin-gonic/gin/binding"

//
// Params
const (
	ID        = "id"
	ID2       = "id2"
	Key       = "key"
	Name      = "name"
	Wildcard  = "wildcard"
	FileField = "file"
	Filter    = "filter"
)

//
// Headers
const (
	Accept        = "Accept"
	Authorization = "Authorization"
	ContentLength = "Content-Length"
	ContentType   = "Content-Type"
	Directory     = "X-Directory"
)

//
// MIME Types.
const (
	MIMEOCTETSTREAM = "application/octet-stream"
)

//
// BindMIMEs supported binding MIME types.
var BindMIMEs = []string{binding.MIMEJSON, binding.MIMEYAML}

//
// Header Values
const (
	DirectoryArchive = "archive"
	DirectoryExpand  = "expand"
)
