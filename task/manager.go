package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	liberr "github.com/jortel/go-utils/error"
	"github.com/jortel/go-utils/logr"
	"github.com/konveyor/tackle2-hub/auth"
	k8s2 "github.com/konveyor/tackle2-hub/k8s"
	crd "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	"github.com/konveyor/tackle2-hub/metrics"
	"github.com/konveyor/tackle2-hub/model"
	"github.com/konveyor/tackle2-hub/settings"
	"gorm.io/gorm"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// States
const (
	Created   = "Created"
	Postponed = "Postponed"
	Ready     = "Ready"
	Pending   = "Pending"
	Running   = "Running"
	Succeeded = "Succeeded"
	Failed    = "Failed"
	Canceled  = "Canceled"
)

// Policies
const (
	Isolated = "isolated"
)

const (
	Unit = time.Second
)

const (
	Shared = "shared"
	Cache  = "cache"
)

var (
	Settings = &settings.Settings
	Log      = logr.WithName("task-scheduler")
)

// AddonNotFound used to report addon referenced
// by a task but cannot be found.
type AddonNotFound struct {
	Name string
}

func (e *AddonNotFound) Error() (s string) {
	return fmt.Sprintf("Addon: '%s' not-found.", e.Name)
}

func (e *AddonNotFound) Is(err error) (matched bool) {
	_, matched = err.(*AddonNotFound)
	return
}

// Manager provides task management.
type Manager struct {
	// DB
	DB *gorm.DB
	// k8s client.
	Client k8s.Client
	// Addon token scopes.
	Scopes []string
}

// Run the manager.
func (m *Manager) Run(ctx context.Context) {
	auth.Validators = append(
		auth.Validators,
		&Validator{
			Client: m.Client,
		})
	go func() {
		Log.Info("Started.")
		defer Log.Info("Done.")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m.updateRunning()
				m.startReady()
				m.pause()
			}
		}
	}()
}

// Pause.
func (m *Manager) pause() {
	d := Unit * time.Duration(Settings.Frequency.Task)
	time.Sleep(d)
}

// startReady starts pending tasks.
func (m *Manager) startReady() {
	list := []model.Task{}
	db := m.DB.Order("priority DESC, id")
	result := db.Find(
		&list,
		"state IN ?",
		[]string{
			Ready,
			Postponed,
			Pending,
			Running,
		})
	Log.Error(result.Error, "")
	if result.Error != nil {
		return
	}
	for i := range list {
		task := &list[i]
		if Settings.Disconnected {
			mark := time.Now()
			task.State = Failed
			task.Terminated = &mark
			task.Error("Error", "Hub is disconnected.")
			sErr := m.DB.Save(task).Error
			Log.Error(sErr, "")
			continue
		}
		if task.Canceled {
			m.canceled(task)
			continue
		}
		switch task.State {
		case Ready,
			Postponed:
			ready := task
			if m.postpone(ready, list) {
				ready.State = Postponed
				Log.Info("Task postponed.", "id", ready.ID)
				sErr := m.DB.Save(ready).Error
				Log.Error(sErr, "")
				continue
			}
			if ready.Retries == 0 {
				metrics.TasksInitiated.Inc()
			}
			rt := Task{ready}
			err := rt.Run(m.Client)
			if err != nil {
				if errors.Is(err, &AddonNotFound{}) {
					ready.Error("Error", err.Error())
					ready.State = Failed
					sErr := m.DB.Save(ready).Error
					Log.Error(sErr, "")
				}
				Log.Error(err, "")
				continue
			}
			Log.Info("Task started.", "id", ready.ID)
			err = m.DB.Save(ready).Error
			Log.Error(err, "")
		default:
			// Ignored.
			// Other states included to support
			// postpone rules.
		}
	}
}

// updateRunning tasks to reflect pod state.
func (m *Manager) updateRunning() {
	list := []model.Task{}
	db := m.DB.Order("priority DESC, id")
	result := db.Find(
		&list,
		"state IN ?",
		[]string{
			Pending,
			Running,
		})
	Log.Error(result.Error, "")
	if result.Error != nil {
		return
	}
	for _, running := range list {
		if running.Canceled {
			m.canceled(&running)
			continue
		}
		rt := Task{&running}
		err := rt.Reflect(m.Client)
		if err != nil {
			Log.Error(err, "")
			continue
		}
		err = m.DB.Save(&running).Error
		if err != nil {
			Log.Error(result.Error, "")
			continue
		}
		Log.V(1).Info("Task updated.", "id", running.ID)
	}
}

// postpone Postpones a task as needed based on rules.
func (m *Manager) postpone(ready *model.Task, list []model.Task) (postponed bool) {
	ruleSet := []Rule{
		&RuleIsolated{},
		&RuleUnique{},
	}
	for i := range list {
		other := &list[i]
		if ready.ID == other.ID {
			continue
		}
		switch other.State {
		case Running,
			Pending:
			for _, rule := range ruleSet {
				if rule.Match(ready, other) {
					postponed = true
					return
				}
			}
		}
	}

	return
}

// The task has been canceled.
func (m *Manager) canceled(task *model.Task) {
	rt := Task{task}
	err := rt.Cancel(m.Client)
	Log.Error(err, "")
	if err != nil {
		return
	}
	err = m.DB.Save(task).Error
	Log.Error(err, "")
	db := m.DB.Model(&model.TaskReport{})
	err = db.Delete("taskid", task.ID).Error
	Log.Error(err, "")
	return
}

// Task is an runtime task.
type Task struct {
	// model.
	*model.Task
}

// Run the specified task.
func (r *Task) Run(client k8s.Client) (err error) {
	mark := time.Now()
	defer func() {
		if err != nil {
			r.Error("Error", err.Error())
			r.Terminated = &mark
			r.State = Failed
		}
	}()
	addon, err := r.findAddon(client, r.Addon)
	if err != nil {
		return
	}
	owner, err := r.findTackle(client)
	if err != nil {
		return
	}
	r.Image = addon.Spec.Image
	secret := r.secret(addon)
	err = client.Create(context.TODO(), &secret)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		if err != nil {
			_ = client.Delete(context.TODO(), &secret)
		}
	}()
	pod := r.pod(addon, owner, &secret)
	if variant, found := Variants[r.Addon]; found {
		err = variant(client, r.Task, &pod.Spec)
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	err = client.Create(context.TODO(), &pod)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		if err != nil {
			_ = client.Delete(context.TODO(), &pod)
		}
	}()
	secret.OwnerReferences = append(
		secret.OwnerReferences,
		meta.OwnerReference{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       pod.Name,
			UID:        pod.UID,
		})
	err = client.Update(context.TODO(), &secret)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	r.Started = &mark
	r.State = Pending
	r.Pod = path.Join(
		pod.Namespace,
		pod.Name)
	return
}

// Reflect finds the associated pod and updates the task state.
func (r *Task) Reflect(client k8s.Client) (err error) {
	pod := &core.Pod{}
	err = client.Get(
		context.TODO(),
		k8s.ObjectKey{
			Namespace: path.Dir(r.Pod),
			Name:      path.Base(r.Pod),
		},
		pod)
	if err != nil {
		if k8serr.IsNotFound(err) {
			err = r.Run(client)
		} else {
			err = liberr.Wrap(err)
		}
		return
	}
	switch pod.Status.Phase {
	case core.PodPending:
		r.podPending(pod)
	case core.PodRunning:
		r.podRunning(pod, client)
	case core.PodSucceeded:
		r.podSucceeded(pod)
	case core.PodFailed:
		r.podFailed(pod, client)
	}

	return
}

// Delete the associated pod as needed.
func (r *Task) Delete(client k8s.Client) (err error) {
	if r.Pod == "" {
		return
	}
	pod := &core.Pod{}
	pod.Namespace = path.Dir(r.Pod)
	pod.Name = path.Base(r.Pod)
	err = client.Delete(context.TODO(), pod)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			err = liberr.Wrap(err)
			return
		} else {
			err = nil
		}
	}
	r.Pod = ""
	Log.Info(
		"Task pod deleted.",
		"id",
		r.ID,
		"pod",
		pod.Name)
	mark := time.Now()
	r.Terminated = &mark
	return
}

// podPending handles pod pending.
func (r *Task) podPending(pod *core.Pod) {
	for _, status := range pod.Status.InitContainerStatuses {
		if status.Started == nil {
			continue
		}
		if *status.Started {
			r.State = Running
			return
		}
	}
}

// podRunning handles pod running.
func (r *Task) podRunning(pod *core.Pod, client k8s.Client) {
	var statuses []core.ContainerStatus
	statuses = append(
		statuses,
		pod.Status.InitContainerStatuses...)
	statuses = append(
		statuses,
		pod.Status.ContainerStatuses...)
	for _, status := range statuses {
		if status.State.Terminated == nil {
			continue
		}
		switch status.State.Terminated.ExitCode {
		case 0: // Succeeded.
		default: // failed.
			r.podFailed(pod, client)
		}
	}
}

// podFailed handles pod succeeded.
func (r *Task) podSucceeded(pod *core.Pod) {
	mark := time.Now()
	r.State = Succeeded
	r.Terminated = &mark

}

// podFailed handles pod failed.
func (r *Task) podFailed(pod *core.Pod, client k8s.Client) {
	mark := time.Now()
	var statuses []core.ContainerStatus
	statuses = append(
		statuses,
		pod.Status.InitContainerStatuses...)
	statuses = append(
		statuses,
		pod.Status.ContainerStatuses...)
	for _, status := range statuses {
		if status.State.Terminated == nil {
			continue
		}
		switch status.State.Terminated.ExitCode {
		case 0: // Succeeded.
		case 137: // Killed.
			if r.Retries < Settings.Hub.Task.Retries {
				_ = client.Delete(context.TODO(), pod)
				r.Pod = ""
				r.State = Ready
				r.Errors = nil
				r.Retries++
				return
			}
			fallthrough
		default: // Error.
			r.State = Failed
			r.Terminated = &mark
			r.Error(
				"Error",
				"Container (%s) failed: %s",
				status.Name,
				status.State.Terminated.Reason)
			return
		}
	}
}

// Cancel the task.
func (r *Task) Cancel(client k8s.Client) (err error) {
	err = r.Delete(client)
	if err != nil {
		return
	}
	r.State = Canceled
	r.SetBucket(nil)
	Log.Info(
		"Task canceled.",
		"id",
		r.ID)
	return
}

// findAddon by name.
func (r *Task) findAddon(client k8s.Client, name string) (addon *crd.Addon, err error) {
	addon = &crd.Addon{}
	err = client.Get(
		context.TODO(),
		k8s.ObjectKey{
			Namespace: Settings.Hub.Namespace,
			Name:      name,
		},
		addon)
	if err != nil {
		if k8serr.IsNotFound(err) {
			err = &AddonNotFound{name}
		} else {
			err = liberr.Wrap(err)
		}
		return
	}

	return
}

// findTackle returns the tackle CR.
func (r *Task) findTackle(client k8s.Client) (owner *crd.Tackle, err error) {
	list := crd.TackleList{}
	err = client.List(
		context.TODO(),
		&list,
		&k8s.ListOptions{Namespace: Settings.Namespace})
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	if len(list.Items) == 0 {
		err = liberr.New("Tackle CR not found.")
		return
	}
	owner = &list.Items[0]
	return
}

// pod build the pod.
func (r *Task) pod(addon *crd.Addon, owner *crd.Tackle, secret *core.Secret) (pod core.Pod) {
	pod = core.Pod{
		Spec: r.specification(addon, secret),
		ObjectMeta: meta.ObjectMeta{
			Namespace:    Settings.Hub.Namespace,
			GenerateName: r.k8sName(),
			Labels:       r.labels(),
		},
	}
	pod.OwnerReferences = append(
		pod.OwnerReferences,
		meta.OwnerReference{
			APIVersion: owner.APIVersion,
			Kind:       owner.Kind,
			Name:       owner.Name,
			UID:        owner.UID,
		})

	return
}

// specification builds a Pod specification.
func (r *Task) specification(addon *crd.Addon, secret *core.Secret) (specification core.PodSpec) {
	sharedProcess := true
	specification.ShareProcessNamespace = &sharedProcess
	shared := core.Volume{
		Name: Shared,
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	}
	cache := core.Volume{
		Name: Cache,
	}
	if Settings.Cache.RWX {
		cache.VolumeSource = core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: Settings.Cache.PVC,
			},
		}
	} else {
		cache.VolumeSource = core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		}
	}
	specification = core.PodSpec{
		ServiceAccountName: Settings.Hub.Task.SA,
		RestartPolicy:      core.RestartPolicyNever,
		Containers: []core.Container{
			r.container(addon, secret),
		},
		Volumes: []core.Volume{
			shared,
			cache,
		},
	}

	return
}

// container builds the pod container.
func (r *Task) container(addon *crd.Addon, secret *core.Secret) (container core.Container) {
	userid := int64(0)
	token := &core.EnvVarSource{
		SecretKeyRef: &core.SecretKeySelector{
			Key: settings.EnvHubToken,
			LocalObjectReference: core.LocalObjectReference{
				Name: secret.Name,
			},
		},
	}
	container.Name = "main"
	container.Image = addon.Spec.Image
	container.ImagePullPolicy = addon.Spec.ImagePullPolicy
	container.SecurityContext = &core.SecurityContext{
		RunAsUser: &userid,
	}
	container.VolumeMounts = append(
		container.VolumeMounts,
		core.VolumeMount{
			Name:      Shared,
			MountPath: Settings.Shared.Path,
		},
		core.VolumeMount{
			Name:      Cache,
			MountPath: Settings.Cache.Path,
		})
	container.Env = append(
		container.Env,
		core.EnvVar{
			Name:  settings.EnvHubBaseURL,
			Value: Settings.Addon.Hub.URL,
		},
		core.EnvVar{
			Name:  settings.EnvTask,
			Value: strconv.Itoa(int(r.Task.ID)),
		},
		core.EnvVar{
			Name:      settings.EnvHubToken,
			ValueFrom: token,
		})
	return
}

// secret builds the pod secret.
func (r *Task) secret(addon *crd.Addon) (secret core.Secret) {
	user := "addon:" + addon.Name
	token, _ := auth.Hub.NewToken(
		user,
		auth.AddonRole,
		jwt.MapClaims{
			"task": r.ID,
		})
	secret = core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Namespace:    Settings.Hub.Namespace,
			GenerateName: r.k8sName(),
			Labels:       r.labels(),
		},
		Data: map[string][]byte{
			settings.EnvHubToken: []byte(token),
		},
	}

	return
}

// k8sName returns a name suitable to be used for k8s resources.
func (r *Task) k8sName() string {
	return fmt.Sprintf("task-%d-", r.ID)
}

// labels builds k8s labels.
func (r *Task) labels() map[string]string {
	return map[string]string{
		"task": strconv.Itoa(int(r.ID)),
		"app":  "tackle",
		"role": "task",
	}
}

// podLog - get and store pod log as a File.
func (r *Task) podLog(db *gorm.DB) (file *model.File, err error) {
	options := &core.PodLogOptions{}
	clientSet, err := k8s2.NewClientSet()
	if err != nil {
		return
	}
	podClient := clientSet.CoreV1().Pods(Settings.Hub.Namespace)
	req := podClient.GetLogs(r.Pod, options)
	reader, err := req.Stream(context.TODO())
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		_ = reader.Close()
	}()
	file = &model.File{Name: "pod.log"}
	err = db.Create(&file).Error
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	writer, err := os.Create(file.Path)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		_ = writer.Close()
	}()
	_, err = io.Copy(writer, reader)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	return
}
