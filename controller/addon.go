package controller

import (
	"context"
	libcnd "github.com/konveyor/controller/pkg/condition"
	"github.com/konveyor/controller/pkg/logging"
	api "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	"github.com/konveyor/tackle2-hub/settings"
	"gorm.io/gorm"
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
	Name = "addon"
)

//
// Package logger.
var log = logging.WithName(Name)

//
// Settings defines applcation settings.
var Settings = &settings.Settings

//
// Add the controller.
func Add(mgr manager.Manager, db *gorm.DB) error {
	reconciler := &Reconciler{
		EventRecorder: mgr.GetRecorder(Name),
		Client:        mgr.GetClient(),
		Log:           log,
		DB:            db,
	}
	cnt, err := controller.New(
		Name,
		mgr,
		controller.Options{
			Reconciler: reconciler,
		})
	if err != nil {
		log.Trace(err)
		return err
	}
	// Primary CR.
	err = cnt.Watch(
		&source.Kind{Type: &api.Addon{}},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		log.Trace(err)
		return err
	}

	return nil
}

//
// Reconciler reconciles addon CRs.
type Reconciler struct {
	record.EventRecorder
	k8s.Client
	DB *gorm.DB
	Log *logging.Logger
}

//
// Reconcile a Addon CR.
// Note: Must not a pointer receiver to ensure that the
// logger and other state is not shared.
func (r Reconciler) Reconcile(request reconcile.Request) (result reconcile.Result, err error) {
	r.Log = logging.WithName(
		names.SimpleNameGenerator.GenerateName(Name+"|"),
		"addon",
		request)

	// Fetch the CR.
	addon := &api.Addon{}
	err = r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		if k8serr.IsNotFound(err) {
			r.Log.Info("Addon deleted.")
			err = nil
		}
		return
	}

	// Begin staging conditions.
	addon.Status.BeginStagingConditions()

	// Ready condition.
	if !addon.Status.HasBlockerCondition() {
		addon.Status.SetCondition(libcnd.Condition{
			Type:     libcnd.Ready,
			Status:   "True",
			Category: "Required",
			Message:  "The addon is ready.",
		})
	}

	// End staging conditions.
	addon.Status.EndStagingConditions()

	// Apply changes.
	addon.Status.ObservedGeneration = addon.Generation
	err = r.Status().Update(context.TODO(), addon)
	if err != nil {
		return
	}

	// Done
	return
}
