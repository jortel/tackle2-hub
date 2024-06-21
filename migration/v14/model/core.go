package model

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	liberr "github.com/jortel/go-utils/error"
	"github.com/konveyor/tackle2-hub/encryption"
	"gorm.io/gorm"
)

// Model Base model.
type Model struct {
	ID         uint      `gorm:"<-:create;primaryKey"`
	CreateTime time.Time `gorm:"<-:create;autoCreateTime"`
	CreateUser string    `gorm:"<-:create"`
	UpdateUser string
}

type Setting struct {
	Model
	Key   string `gorm:"<-:create;uniqueIndex"`
	Value JSON   `gorm:"type:json"`
}

// With updates the value of the Setting with the json representation
// of the `value` parameter.
func (r *Setting) With(value any) (err error) {
	r.Value, err = json.Marshal(value)
	if err != nil {
		err = liberr.Wrap(err)
	}
	return
}

// As unmarshalls the value of the Setting into the `ptr` parameter.
func (r *Setting) As(ptr any) (err error) {
	err = json.Unmarshal(r.Value, ptr)
	if err != nil {
		err = liberr.Wrap(err)
	}
	return
}

type Bucket struct {
	Model
	Path       string `gorm:"<-:create;uniqueIndex"`
	Expiration *time.Time
}

func (m *Bucket) BeforeCreate(db *gorm.DB) (err error) {
	if m.Path == "" {
		uid := uuid.New()
		m.Path = path.Join(
			Settings.Hub.Bucket.Path,
			uid.String())
		err = os.MkdirAll(m.Path, 0777)
		if err != nil {
			err = liberr.Wrap(
				err,
				"path",
				m.Path)
		}
	}
	return
}

type BucketOwner struct {
	BucketID *uint `gorm:"index" ref:"bucket"`
	Bucket   *Bucket
}

func (m *BucketOwner) BeforeCreate(db *gorm.DB) (err error) {
	if !m.HasBucket() {
		b := &Bucket{}
		err = db.Create(b).Error
		m.SetBucket(&b.ID)
	}
	return
}

func (m *BucketOwner) SetBucket(id *uint) {
	m.BucketID = id
	m.Bucket = nil
}

func (m *BucketOwner) HasBucket() (b bool) {
	return m.BucketID != nil
}

type File struct {
	Model
	Name       string
	Path       string `gorm:"<-:create;uniqueIndex"`
	Expiration *time.Time
}

func (m *File) BeforeCreate(db *gorm.DB) (err error) {
	uid := uuid.New()
	m.Path = path.Join(
		Settings.Hub.Bucket.Path,
		".file",
		uid.String())
	err = os.MkdirAll(path.Dir(m.Path), 0777)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			m.Path)
	}
	return
}

type Task struct {
	Model
	BucketOwner
	Name          string `gorm:"index"`
	Kind          string
	Addon         string   `gorm:"index"`
	Extensions    []string `gorm:"type:json;serializer:json"`
	State         string   `gorm:"index"`
	Locator       string   `gorm:"index"`
	Priority      int
	Policy        TaskPolicy `gorm:"type:json;serializer:json"`
	TTL           TTL        `gorm:"type:json;serializer:json"`
	Data          Data       `gorm:"type:json;serializer:json"`
	Started       *time.Time
	Terminated    *time.Time
	Errors        []TaskError `gorm:"type:json;serializer:json"`
	Events        []TaskEvent `gorm:"type:json;serializer:json"`
	Pod           string      `gorm:"index"`
	Retries       int
	Attached      []Attachment `gorm:"type:json;serializer:json" ref:"[]file"`
	Report        *TaskReport  `gorm:"constraint:OnDelete:CASCADE"`
	ApplicationID *uint
	Application   *Application
	TaskGroupID   *uint `gorm:"<-:create"`
	TaskGroup     *TaskGroup
}

func (m *Task) BeforeCreate(db *gorm.DB) (err error) {
	err = m.BucketOwner.BeforeCreate(db)
	return
}

// TaskError used in Task.Errors.
type TaskError struct {
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// TaskPolicy scheduling policy.
type TaskPolicy struct {
	Isolated       bool `json:"isolated,omitempty" yaml:",omitempty"`
	PreemptEnabled bool `json:"preemptEnabled,omitempty" yaml:"preemptEnabled,omitempty"`
	PreemptExempt  bool `json:"preemptExempt,omitempty" yaml:"preemptExempt,omitempty"`
}

// Attachment file attachment.
type Attachment struct {
	ID       uint   `json:"id" binding:"required"`
	Name     string `json:"name,omitempty" yaml:",omitempty"`
	Activity int    `json:"activity,omitempty" yaml:",omitempty"`
}

type TaskReport struct {
	Model
	Status    string
	Total     int
	Completed int
	Activity  []string     `gorm:"type:json;serializer:json"`
	Errors    []TaskError  `gorm:"type:json;serializer:json"`
	Attached  []Attachment `gorm:"type:json;serializer:json" ref:"[]file"`
	Result    Data         `gorm:"type:json;serializer:json"`
	TaskID    uint         `gorm:"<-:create;uniqueIndex"`
	Task      *Task
}

type TaskGroup struct {
	Model
	BucketOwner
	Name       string
	Kind       string
	Addon      string
	Extensions []string `gorm:"type:json;serializer:json"`
	State      string
	Priority   int
	Policy     TaskPolicy `gorm:"type:json;serializer:json"`
	Data       Data       `gorm:"type:json;serializer:json"`
	List       []Task     `gorm:"type:json;serializer:json"`
	Tasks      []Task     `gorm:"constraint:OnDelete:CASCADE"`
}

// Proxy configuration.
// kind = (http|https)
type Proxy struct {
	Model
	Enabled    bool
	Kind       string `gorm:"uniqueIndex"`
	Host       string `gorm:"not null"`
	Port       int
	Excluded   JSON  `gorm:"type:json"`
	IdentityID *uint `gorm:"index"`
	Identity   *Identity
}

// Identity represents and identity with a set of credentials.
type Identity struct {
	Model
	Kind         string `gorm:"not null"`
	Name         string `gorm:"index;unique;not null"`
	Description  string
	User         string
	Password     string
	Key          string
	Settings     string
	Proxies      []Proxy       `gorm:"constraint:OnDelete:SET NULL"`
	Applications []Application `gorm:"many2many:ApplicationIdentity;constraint:OnDelete:CASCADE"`
}

// Encrypt sensitive fields.
// The ref identity is used to determine when sensitive fields
// have changed and need to be (re)encrypted.
func (r *Identity) Encrypt(ref *Identity) (err error) {
	passphrase := Settings.Encryption.Passphrase
	aes := encryption.New(passphrase)
	if r.Password != ref.Password {
		if r.Password != "" {
			r.Password, err = aes.Encrypt(r.Password)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}
		}
	}
	if r.Key != ref.Key {
		if r.Key != "" {
			r.Key, err = aes.Encrypt(r.Key)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}
		}
	}
	if r.Settings != ref.Settings {
		if r.Settings != "" {
			r.Settings, err = aes.Encrypt(r.Settings)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}
		}
	}
	return
}

// Decrypt sensitive fields.
func (r *Identity) Decrypt() (err error) {
	passphrase := Settings.Encryption.Passphrase
	aes := encryption.New(passphrase)
	if r.Password != "" {
		r.Password, err = aes.Decrypt(r.Password)
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	if r.Key != "" {
		r.Key, err = aes.Decrypt(r.Key)
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	if r.Settings != "" {
		r.Settings, err = aes.Decrypt(r.Settings)
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	return
}

//
// JSON Fields.
//

// TaskEvent task event.
type TaskEvent struct {
	Kind   string    `json:"kind"`
	Count  int       `json:"count"`
	Reason string    `json:"reason,omitempty" yaml:",omitempty"`
	Last   time.Time `json:"last"`
}

// TTL time-to-live.
type TTL struct {
	Created   int `json:"created,omitempty" yaml:",omitempty"`
	Pending   int `json:"pending,omitempty" yaml:",omitempty"`
	Running   int `json:"running,omitempty" yaml:",omitempty"`
	Succeeded int `json:"succeeded,omitempty" yaml:",omitempty"`
	Failed    int `json:"failed,omitempty" yaml:",omitempty"`
}

// Ref represents a FK.
type Ref struct {
	ID   uint   `json:"id" binding:"required"`
	Name string `json:"name,omitempty" yaml:",omitempty"`
}

// Map alias.
type Map = map[string]any

// Any alias.
type Any any

// Data json any field.
type Data struct {
	Any
}

// Merge merges the other into self.
// Both must be a map.
func (d *Data) Merge(other Data) (merged bool) {
	b, isMap := d.AsMap()
	if !isMap {
		return
	}
	a, isMap := other.AsMap()
	if !isMap {
		return
	}
	d.Any = d.merge(a, b)
	merged = true
	return
}

// Merge maps B into A.
// The B map takes precedence.
func (d *Data) merge(a, b map[string]any) (out map[string]any) {
	if a == nil {
		a = make(map[string]any)
	}
	if b == nil {
		b = make(map[string]any)
	}
	out = make(map[string]any)
	for k, v := range a {
		out[k] = v
		if bv, found := b[k]; found {
			out[k] = bv
			if av, cast := v.(map[string]any); cast {
				if bv, cast := bv.(map[string]any); cast {
					out[k] = d.merge(av, bv)
				} else {
					out[k] = bv
				}
			}
		}
	}
	for k, v := range b {
		if _, found := a[k]; !found {
			out[k] = v
		}
	}

	return
}

// AsMap returns self as a map.
func (d *Data) AsMap() (mp map[string]any, isMap bool) {
	if d.Any == nil {
		return
	}
	b, _ := json.Marshal(d.Any)
	mp = make(map[string]any)
	err := json.Unmarshal(b, &mp)
	isMap = err == nil
	return
}
