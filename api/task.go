package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
	"time"
)

//
// Routes
const (
	TasksRoot             = "/tasks"
	TaskRoot              = TasksRoot + "/:" + ID
	TaskReportRoot        = TaskRoot + "/report"
	TaskBucketRoot        = TaskRoot + "/bucket"
	TaskBucketContentRoot = TaskBucketRoot + "/*" + Wildcard
	TaskSubmitRoot        = TaskRoot + "/submit"
	TaskCancelRoot        = TaskRoot + "/cancel"
)

//
// TTL time-to-live.
type TTL struct {
	Created   int `json:"created,omitempty"`
	Pending   int `json:"pending,omitempty"`
	Postponed int `json:"postponed,omitempty"`
	Running   int `json:"running,omitempty"`
	Succeeded int `json:"succeeded,omitempty"`
	Failed    int `json:"failed,omitempty"`
}

//
// Task REST resource.
type Task struct {
	Resource
	Name        string      `json:"name"`
	Locator     string      `json:"locator,omitempty"`
	Priority    int         `json:"priority,omitempty"`
	Variant     string      `json:"variant,omitempty"`
	Policy      string      `json:"policy,omitempty"`
	TTL         *TTL        `json:"ttl,omitempty"`
	Addon       string      `json:"addon,omitempty" binding:"required"`
	Data        interface{} `json:"data" swaggertype:"object" binding:"required"`
	Application *Ref        `json:"application,omitempty"`
	State       string      `json:"state"`
	Image       string      `json:"image,omitempty"`
	Bucket      *Ref        `json:"bucket,omitempty"`
	Purged      bool        `json:"purged,omitempty"`
	Started     *time.Time  `json:"started,omitempty"`
	Terminated  *time.Time  `json:"terminated,omitempty"`
	Error       string      `json:"error,omitempty"`
	Pod         string      `json:"pod,omitempty"`
	Retries     int         `json:"retries,omitempty"`
	Canceled    bool        `json:"canceled,omitempty"`
	Report      *TaskReport `json:"report,omitempty"`
}

//
// With updates the resource with the model.
func (r *Task) With(m *model.Task) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.Image = m.Image
	r.Addon = m.Addon
	r.Locator = m.Locator
	r.Priority = m.Priority
	r.Policy = m.Policy
	r.Variant = m.Variant
	r.Application = r.refPtr(m.ApplicationID, m.Application)
	r.Bucket = r.refPtr(m.BucketID, m.Bucket)
	r.State = m.State
	r.Started = m.Started
	r.Terminated = m.Terminated
	r.Error = m.Error
	r.Pod = m.Pod
	r.Retries = m.Retries
	r.Canceled = m.Canceled
	_ = json.Unmarshal(m.Data, &r.Data)
	if m.Report != nil {
		report := &TaskReport{}
		report.With(m.Report)
		r.Report = report
	}
	if m.TTL != nil {
		_ = json.Unmarshal(m.TTL, &r.TTL)
	}
}

//
// Model builds a model.
func (r *Task) Model() (m *model.Task) {
	m = &model.Task{
		Name:          r.Name,
		Addon:         r.Addon,
		Locator:       r.Locator,
		Variant:       r.Variant,
		Priority:      r.Priority,
		Policy:        r.Policy,
		State:         r.State,
		ApplicationID: r.idPtr(r.Application),
	}
	m.Data, _ = json.Marshal(r.Data)
	m.ID = r.ID
	if r.TTL != nil {
		m.TTL, _ = json.Marshal(r.TTL)
	}
	return
}

//
// TaskReport REST resource.
type TaskReport struct {
	Resource
	Status    string      `json:"status"`
	Error     string      `json:"error"`
	Total     int         `json:"total"`
	Completed int         `json:"completed"`
	Activity  []string    `json:"activity"`
	Result    interface{} `json:"result,omitempty" swaggertype:"object"`
	TaskID    uint        `json:"task"`
}

//
// With updates the resource with the model.
func (r *TaskReport) With(m *model.TaskReport) {
	r.Resource.With(&m.Model)
	r.Status = m.Status
	r.Error = m.Error
	r.Total = m.Total
	r.Completed = m.Completed
	r.TaskID = m.TaskID
	if m.Activity != nil {
		_ = json.Unmarshal(m.Activity, &r.Activity)
	}
	if m.Result != nil {
		_ = json.Unmarshal(m.Result, &r.Result)
	}
}

//
// Model builds a model.
func (r *TaskReport) Model() (m *model.TaskReport) {
	if r.Activity == nil {
		r.Activity = []string{}
	}
	m = &model.TaskReport{
		Status:    r.Status,
		Error:     r.Error,
		Total:     r.Total,
		Completed: r.Completed,
		TaskID:    r.TaskID,
	}
	if r.Activity != nil {
		m.Activity, _ = json.Marshal(r.Activity)
	}
	if r.Result != nil {
		m.Result, _ = json.Marshal(r.Result)
	}
	m.ID = r.ID

	return
}
