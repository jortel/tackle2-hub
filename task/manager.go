package task

import (
	"context"
	"encoding/json"
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
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TaskLabel    = "konveyor.io/task"
	AddonInit    = "init"
	AddonDaemon  = "daemon"
	AddonCommand = "command"
	AddonKindEnv = "ADDON_KIND"
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

// NotFound used to report kind referenced
// by a task but cannot be found.
type NotFound struct {
	Name string
}

func (e *NotFound) Error() (s string) {
	return fmt.Sprintf("Task (kind): '%s' not-found.", e.Name)
}

func (e *NotFound) Is(err error) (matched bool) {
	_, matched = err.(*NotFound)
	return
}

// NoAddonsMatched used to report task kind not matched to addons.
type NoAddonsMatched struct {
	Task string
}

func (e *NoAddonsMatched) Error() (s string) {
	return fmt.Sprintf("Task: '%s' no kind=%s addons matched.", AddonCommand, e.Task)
}

func (e *NoAddonsMatched) Is(err error) (matched bool) {
	_, matched = err.(*NoAddonsMatched)
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

func (m *Manager) db() (db *gorm.DB) {
	db = m.DB.Session(&gorm.Session{})
	return
}

// Pause.
func (m *Manager) pause() {
	d := Unit * time.Duration(Settings.Frequency.Task)
	time.Sleep(d)
}

// startReady starts pending tasks.
func (m *Manager) startReady() {
	list := []model.Task{}
	db := m.db().Order("priority DESC, id")
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
			sErr := m.db().Save(task).Error
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
				sErr := m.db().Save(ready).Error
				Log.Error(sErr, "")
				continue
			}
			if ready.Retries == 0 {
				metrics.TasksInitiated.Inc()
			}
			rt := Task{ready}
			err := rt.Run(m.Client)
			if err != nil {
				ready.State = Failed
				Log.Error(err, "")
			} else {
				Log.Info("Task started.", "id", ready.ID)
			}
			err = m.db().Save(ready).Error
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
	db := m.db().Order("priority DESC, id")
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
		pod, err := rt.Reflect(m.Client)
		if err != nil {
			Log.Error(err, "")
			continue
		}
		if rt.State == Succeeded || rt.State == Failed {
			err = m.snapshotPod(&rt, pod)
			if err != nil {
				Log.Error(err, "")
				continue
			}
			err = rt.Delete(m.Client)
			if err != nil {
				Log.Error(err, "")
				continue
			}
		}
		err = m.db().Save(&running).Error
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
	err = m.db().Save(task).Error
	Log.Error(err, "")
	db := m.db().Model(&model.TaskReport{})
	err = db.Delete("taskid", task.ID).Error
	Log.Error(err, "")
	return
}

// snapshotPod attaches a pod description and logs.
func (m *Manager) snapshotPod(task *Task, pod *core.Pod) (err error) {
	var files []*model.File
	d, err := m.podDescription(pod)
	if err != nil {
		return
	}
	files = append(files, d)
	logs, err := m.podLogs(pod)
	if err != nil {
		return
	}
	files = append(files, logs...)
	for _, f := range files {
		task.attach(f)
	}
	Log.V(1).Info("Task pod snapshot attached.", "id", task.ID)
	return
}

// podDescription builds pod resource description.
func (m *Manager) podDescription(pod *core.Pod) (file *model.File, err error) {
	file = &model.File{Name: "pod.yaml"}
	err = m.db().Create(file).Error
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	f, err := os.Create(file.Path)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		_ = f.Close()
	}()
	b, _ := yaml.Marshal(pod)
	_, _ = f.Write(b)
	return
}

// podLogs - get and store pod logs as a Files.
func (m *Manager) podLogs(pod *core.Pod) (files []*model.File, err error) {
	for _, container := range pod.Spec.Containers {
		f, nErr := m.containerLog(pod, container.Name)
		if nErr == nil {
			files = append(files, f)
		} else {
			err = nErr
			return
		}
	}
	return
}

// containerLog - get container log and store in file.
func (m *Manager) containerLog(pod *core.Pod, container string) (file *model.File, err error) {
	options := &core.PodLogOptions{
		Container: container,
	}
	clientSet, err := k8s2.NewClientSet()
	if err != nil {
		return
	}
	podClient := clientSet.CoreV1().Pods(Settings.Hub.Namespace)
	req := podClient.GetLogs(pod.Name, options)
	reader, err := req.Stream(context.TODO())
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		_ = reader.Close()
	}()
	file = &model.File{Name: container + ".log"}
	err = m.db().Create(file).Error
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	f, err := os.Create(file.Path)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = io.Copy(f, reader)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
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
	owner, err := r.findTackle(client)
	if err != nil {
		return
	}
	addons, err := r.findAddons(client)
	if err != nil {
		return
	}
	secret := r.secret()
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
	pod := r.pod(addons, owner, &secret)
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
func (r *Task) Reflect(client k8s.Client) (pod *core.Pod, err error) {
	pod = &core.Pod{}
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
	err = client.Delete(context.TODO(), pod, k8s.GracePeriodSeconds(0))
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
	var status []core.ContainerStatus
	status = append(
		status,
		pod.Status.InitContainerStatuses...)
	status = append(
		status,
		pod.Status.ContainerStatuses...)
	for _, status := range status {
		if status.Started == nil {
			continue
		}
		if *status.Started {
			r.State = Running
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

// podRunning handles pod running.
func (r *Task) podRunning(pod *core.Pod, client k8s.Client) {
	r.State = Running
	command := map[string]bool{}
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == AddonKindEnv && env.Value == AddonCommand {
				command[container.Name] = false
			}
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated != nil {
			switch status.State.Terminated.ExitCode {
			case 0:
				if _, found := command[status.Name]; found {
					command[status.Name] = true
				}
			default: // failed.
				r.podFailed(pod, client)
				return
			}
		}
	}
	n := len(command)
	for _, succeeded := range command {
		if succeeded {
			n--
		}
	}
	if n == 0 {
		r.podSucceeded(pod)
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

// findAddon by name.
func (r *Task) findAddons(client k8s.Client) (addons []crd.Addon, err error) {
	def := &crd.Task{}
	err = client.Get(
		context.TODO(),
		k8s.ObjectKey{
			Namespace: Settings.Hub.Namespace,
			Name:      r.Kind,
		},
		def)
	if err != nil {
		if k8serr.IsNotFound(err) {
			err = &NotFound{r.Kind}
		} else {
			err = liberr.Wrap(err)
		}
		return
	}
	addonList := crd.AddonList{}
	err = client.List(
		context.TODO(),
		&addonList,
		k8s.InNamespace(Settings.Hub.Namespace),
		k8s.MatchingLabels{
			TaskLabel: r.Kind,
		})
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for _, addon := range addonList.Items {
		if addon.Spec.Kind == AddonCommand {
			addons = addonList.Items
			return
		}
	}
	err = &NoAddonsMatched{Task: r.Name}
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
func (r *Task) pod(addons []crd.Addon, owner *crd.Tackle, secret *core.Secret) (pod core.Pod) {
	pod = core.Pod{
		Spec: r.specification(addons, secret),
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
func (r *Task) specification(addons []crd.Addon, secret *core.Secret) (specification core.PodSpec) {
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
	init, plain := r.containers(addons, secret)
	specification = core.PodSpec{
		ServiceAccountName: Settings.Hub.Task.SA,
		RestartPolicy:      core.RestartPolicyNever,
		InitContainers:     init,
		Containers:         plain,
		Volumes: []core.Volume{
			shared,
			cache,
		},
	}

	return
}

// container builds the pod containers.
func (r *Task) containers(
	addons []crd.Addon,
	secret *core.Secret) (init []core.Container, plain []core.Container) {
	userid := int64(0)
	token := &core.EnvVarSource{
		SecretKeyRef: &core.SecretKeySelector{
			Key: settings.EnvHubToken,
			LocalObjectReference: core.LocalObjectReference{
				Name: secret.Name,
			},
		},
	}
	for i := range addons {
		addon := addons[i]
		container := addon.Spec.Container
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
				Name:  AddonKindEnv,
				Value: addon.Spec.Kind,
			},
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
		switch addon.Spec.Kind {
		case AddonInit:
			init = append(init, container)
		case AddonDaemon:
			plain = append(plain, container)
		case AddonCommand:
			plain = append(plain, container)
		}
	}
	return
}

// secret builds the pod secret.
func (r *Task) secret() (secret core.Secret) {
	user := "task:" + r.Kind
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

// attach file.
func (r *Task) attach(file *model.File) {
	attached := []model.Ref{}
	_ = json.Unmarshal(r.Attached, &attached)
	attached = append(
		attached,
		model.Ref{
			ID:   file.ID,
			Name: file.Name,
		})
	r.Attached, _ = json.Marshal(attached)
}
