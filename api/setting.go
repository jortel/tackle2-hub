package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	SettingsRoot = "/settings"
	SettingRoot  = SettingsRoot + "/:" + Key
)

//
// Setting REST Resource
type Setting struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (r *Setting) With(m *model.Setting) {
	r.Key = m.Key
	_ = json.Unmarshal(m.Value, &r.Value)

}

func (r *Setting) Model() (m *model.Setting) {
	m = &model.Setting{Key: r.Key}
	m.Value, _ = json.Marshal(r.Value)
	return
}
