package task

import (
	"github.com/konveyor/tackle2-hub/model"
	core "k8s.io/api/core/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

type Variant func(
	client k8s.Client,
	task *model.Task,
	pod *core.PodSpec) (err error)

var Variants = map[string]Variant{}
