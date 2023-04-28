package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
	tasking "github.com/konveyor/tackle2-hub/task"
)

//
// Routes
const (
	TaskGroupsRoot             = "/taskgroups"
	TaskGroupRoot              = TaskGroupsRoot + "/:" + ID
	TaskGroupBucketRoot        = TaskGroupRoot + "/bucket"
	TaskGroupBucketContentRoot = TaskGroupBucketRoot + "/*" + Wildcard
	TaskGroupSubmitRoot        = TaskGroupRoot + "/submit"
)

//
// TaskGroup REST resource.
type TaskGroup struct {
	Resource
	Name   string      `json:"name"`
	Addon  string      `json:"addon"`
	Data   interface{} `json:"data" swaggertype:"object" binding:"required"`
	Bucket *Ref        `json:"bucket,omitempty"`
	State  string      `json:"state"`
	Tasks  []Task      `json:"tasks"`
}

//
// With updates the resource with the model.
func (r *TaskGroup) With(m *model.TaskGroup) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.Addon = m.Addon
	r.State = m.State
	r.Bucket = r.refPtr(m.BucketID, m.Bucket)
	r.Tasks = []Task{}
	_ = json.Unmarshal(m.Data, &r.Data)
	switch m.State {
	case "", tasking.Created:
		_ = json.Unmarshal(m.List, &r.Tasks)
	default:
		for _, task := range m.Tasks {
			member := Task{}
			member.With(&task)
			r.Tasks = append(
				r.Tasks,
				member)
		}
	}
}

//
// Model builds a model.
func (r *TaskGroup) Model() (m *model.TaskGroup) {
	m = &model.TaskGroup{
		Name:  r.Name,
		Addon: r.Addon,
		State: r.State,
	}
	m.ID = r.ID
	m.Data, _ = json.Marshal(r.Data)
	m.List, _ = json.Marshal(r.Tasks)
	if r.Bucket != nil {
		m.BucketID = &r.Bucket.ID
	}
	for _, task := range r.Tasks {
		member := task.Model()
		m.Tasks = append(
			m.Tasks,
			*member)
	}
	return
}
