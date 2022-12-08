package settings

import (
	"fmt"
	"os"
	"strconv"
)

var Settings TackleSettings

type TackleSettings struct {
	Hub
	Metrics
	Addon
	Auth
}

func (r *TackleSettings) Load() (err error) {
	err = r.Hub.Load()
	if err != nil {
		return
	}
	err = r.Addon.Load()
	if err != nil {
		return
	}
	err = r.Auth.Load()
	if err != nil {
		return
	}
	return
}

//
// SettingError used to report invalid settings.
type SettingError struct {
	Name   string
	Reason string
}

func (e *SettingError) Error() (s string) {
	s = fmt.Sprintf(
		"Setting %s not valid. %s",
		e.Name,
		e.Reason)
	return
}
func (e *SettingError) Is(err error) (matched bool) {
	_, matched = err.(*SettingError)
	return
}

//
// Get boolean.
func getEnvBool(name string, def bool) bool {
	boolean := def
	if s, found := os.LookupEnv(name); found {
		parsed, err := strconv.ParseBool(s)
		if err == nil {
			boolean = parsed
		}
	}

	return boolean
}
