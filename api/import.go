package api

import (
	"github.com/konveyor/tackle2-hub/model"
	"time"
)

//
// Record types
const (
	RecordTypeApplication = "1"
	RecordTypeDependency  = "2"
)

//
// Import Statuses
const (
	InProgress = "In Progress"
	Completed  = "Completed"
)

//
// Routes
const (
	SummariesRoot = "/importsummaries"
	SummaryRoot   = SummariesRoot + "/:" + ID
	UploadRoot    = SummariesRoot + "/upload"
	DownloadRoot  = SummariesRoot + "/download"
	ImportsRoot   = "/imports"
	ImportRoot    = ImportsRoot + "/:" + ID
)

//
// Import REST resource.
type Import map[string]interface{}

//
// ImportSummary REST resource.
type ImportSummary struct {
	Resource
	Filename       string    `json:"filename"`
	ImportStatus   string    `json:"importStatus"`
	ImportTime     time.Time `json:"importTime"`
	ValidCount     int       `json:"validCount"`
	InvalidCount   int       `json:"invalidCount"`
	CreateEntities bool      `json:"createEntities"`
}

//
// With updates the resource with the model.
func (r *ImportSummary) With(m *model.ImportSummary) {
	r.Resource.With(&m.Model)
	r.Filename = m.Filename
	r.ImportTime = m.CreateTime
	r.CreateEntities = m.CreateEntities
	for _, imp := range m.Imports {
		if imp.Processed {
			if imp.IsValid {
				r.ValidCount++
			} else {
				r.InvalidCount++
			}
		}
	}
	if len(m.Imports) == r.ValidCount+r.InvalidCount {
		r.ImportStatus = Completed
	} else {
		r.ImportStatus = InProgress
	}
}
