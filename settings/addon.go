package settings

import (
	"net/url"
	"os"
	"strconv"
)

const (
	EnvAddonWorkingDir = "ADDON_WORKING_DIR"
	EnvHubBaseURL      = "HUB_BASE_URL"
	EnvHubTlsEnabled   = "HUB_TLS_ENABLED"
	EnvHubTlsCA        = "HUB_TLS_CA"
	EnvHubToken        = "TOKEN"
	EnvTask            = "TASK"
)

//
// Addon settings.
type Addon struct {
	// Hub settings.
	Hub struct {
		// URL for the hub API.
		URL string
		// Token for the hub API.
		Token string
		// API TLS settings.
		TLS struct {
			Enabled bool
			CA      string
		}
	}
	// Path.
	Path struct {
		// Working directory path.
		WorkingDir string
	}
	//
	Task int
}

func (r *Addon) Load() (err error) {
	var found bool
	r.Hub.URL, found = os.LookupEnv(EnvHubBaseURL)
	if !found {
		r.Hub.URL = "http://localhost:8080"
	}
	_, err = url.Parse(r.Hub.URL)
	if err != nil {
		panic(err)
	}
	r.Hub.Token, _ = os.LookupEnv(EnvHubToken)
	r.Path.WorkingDir, found = os.LookupEnv(EnvAddonWorkingDir)
	if !found {
		r.Path.WorkingDir = "/tmp"
	}
	if s, found := os.LookupEnv(EnvTask); found {
		r.Task, _ = strconv.Atoi(s)
	}
	if s, found := os.LookupEnv(EnvHubTlsEnabled); found {
		r.Hub.TLS.Enabled, _ = strconv.ParseBool(s)
	}
	r.Hub.TLS.CA, _ = os.LookupEnv(EnvHubTlsCA)

	return
}
