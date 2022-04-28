package controller

import (
	"context"
	"fmt"
	libcnd "github.com/konveyor/controller/pkg/condition"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/controller/pkg/logging"
	api "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	"github.com/konveyor/tackle2-hub/settings"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// Name.
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
func Add(mgr manager.Manager) error {
	reconciler := &Reconciler{
		EventRecorder: mgr.GetRecorder(Name),
		Client:        mgr.GetClient(),
		Log:           log,
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

var _ reconcile.Reconciler = &Reconciler{}

//
// Reconciler reconciles addon CRs.
type Reconciler struct {
	record.EventRecorder
	client.Client
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

func (r *Reconciler) ensureClaims(addon *api.Addon) (err error) {
	for _, mount := range addon.Spec.Mounts {
		if mount.Claim != "" {
			continue
		}
		claim := r.claim(addon, &mount)
		err := r.Get(
			context.TODO(),
			client.ObjectKey{
				Namespace: claim.Namespace,
				Name: claim.Name,
			},
			claim)
		if err != nil {
			if !k8serr.IsNotFound(err) {
				err = liberr.Wrap(err)
				return
			} else {
				continue
			}
		}
		err = r.Create(context.TODO(), claim)
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	return
}

func (r *Reconciler) claim(addon *api.Addon, mount *api.Mount) (claim *core.PersistentVolumeClaim) {
	claim = &core.PersistentVolumeClaim{
		ObjectMeta: meta.ObjectMeta{
			Name:      addon.Name + "-" + mount.Name,
			Namespace: addon.Namespace,
		},
		Spec: core.PersistentVolumeClaimSpec{
			Resources: *mount.Capacity,
			StorageClassName: &mount.StorageClass,
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteMany,
			},
		},
	}

	return
}