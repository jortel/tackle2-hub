package pod

import (
	"context"
	"github.com/go-logr/logr"
	logr2 "github.com/jortel/go-utils/logr"
	"github.com/konveyor/tackle2-hub/settings"
	"github.com/konveyor/tackle2-hub/task"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/tools/record"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	Name = "task/pod"
)

//
// Package logger.
var log = logr2.WithName(Name)

//
// Settings defines applcation settings.
var Settings = &settings.Settings

//
// Add the controller.
func Add(owner manager.Manager, taskManager *task.Manager) error {
	reconciler := &Reconciler{
		Client:      owner.GetClient(),
		TaskManager: taskManager,
		Log:         log,
	}
	cnt, err := controller.New(
		Name,
		owner,
		controller.Options{
			Reconciler: reconciler,
		})
	if err != nil {
		log.Error(err, "")
		return err
	}
	// Primary CR.
	err = cnt.Watch(
		&source.Kind{Type: &core.Pod{}},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "")
		return err
	}

	return nil
}

//
// Reconciler reconciles pod CRs.
type Reconciler struct {
	record.EventRecorder
	k8s.Client
	Log         logr.Logger
	TaskManager *task.Manager
}

//
// Reconcile a Pod CR.
// Note: Must not a pointer receiver to ensure that the
// logger and other state is not shared.
func (r Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, err error) {
	r.Log = logr2.WithName(
		names.SimpleNameGenerator.GenerateName(Name+"|"),
		Name,
		request)

	// Fetch the CR.
	pod := &core.Pod{}
	err = r.Get(context.TODO(), request.NamespacedName, pod)
	if err != nil {
		if k8serr.IsNotFound(err) {
			_ = r.TaskManager.PodDeleted(request.Name)
			err = nil
		}
		return
	}

	//
	// changed.
	err = r.TaskManager.PodChanged(pod)
	if err != nil {
		return
	}

	return
}
