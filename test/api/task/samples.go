package task

import (
	"github.com/konveyor/tackle2-hub/api"
)

// Set of valid resources for tests and reuse.
var (
	Windup = api.Task{
		Kind: "test",
		Name: "Test windup task",
		Data: "{}",
	}
	Samples = []api.Task{Windup}
)
