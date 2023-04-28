package api

import (
	"github.com/konveyor/tackle2-hub/model"
	"time"
)

//
// Routes
const (
	BucketsRoot       = "/buckets"
	BucketRoot        = BucketsRoot + "/:" + ID
	BucketContentRoot = BucketRoot + "/*" + Wildcard
)

//
// Bucket REST resource.
type Bucket struct {
	Resource
	Path       string     `json:"path"`
	Expiration *time.Time `json:"expiration,omitempty"`
}

//
// With updates the resource with the model.
func (r *Bucket) With(m *model.Bucket) {
	r.Resource.With(&m.Model)
	r.Path = m.Path
	r.Expiration = m.Expiration
}
