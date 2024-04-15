package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
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
	sched "k8s.io/api/scheduling/v1"
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

// Manager provides task management.
type Manager struct {
	// DB
	DB *gorm.DB
	// k8s client.
	Client k8s.Client
	// Addon token scopes.
	Scopes []string
	// cluster resources.
	cluster Cluster
}

// Run the manager.
func (m *Manager) Run(ctx context.Context) {
	m.cluster.Client = m.Client
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
				err := m.cluster.Refresh()
				if err == nil {
					m.updateRunning()
					m.startReady()
					m.pause()
				} else {
					Log.Error(err, "")
					m.pause()
				}
			}
		}
	}()
}

// Pause.
func (m *Manager) pause() {
	d := Unit * time.Duration(Settings.Frequency.Task)
	time.Sleep(d)
}

// startReady starts ready tasks.
func (m *Manager) startReady() {
	var err error
	defer func() {
		Log.Error(err, "")
	}()
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
	if result.Error != nil {
		return
	}
	err = m.disconnected(list)
	if err != nil {
		return
	}
	err = m.canceled(list)
	if err != nil {
		return
	}
	err = m.adjustPriority(list)
	if err != nil {
		return
	}
	err = m.postpone(list)
	if err != nil {
		return
	}
	sort.Slice(
		list,
		func(i, j int) bool {
			it := &list[i]
			jt := &list[j]
			return it.Priority > jt.Priority ||
				it.ID < jt.ID
		})
	for i := range list {
		task := &list[i]
		if task.State != Ready {
			continue
		}
		ready := task
		rt := Task{ready}
		err := rt.Run(m.DB, m.cluster)
		if err != nil {
			if errors.Is(err, &QuotaExceeded{}) {
				Log.V(1).Info(err.Error())
				err = nil
			} else {
				ready.State = Failed
				Log.Error(err, "")
				err = nil
			}
		} else {
			Log.Info("Task started.", "id", ready.ID)
			if ready.Retries == 0 {
				metrics.TasksInitiated.Inc()
			}
		}
		err = db.Save(ready).Error
		if err != nil {
			return
		}
	}
	return
}

// disconnected fails tasks when hub is disconnected.
func (m *Manager) disconnected(list []model.Task) (err error) {
	if !Settings.Disconnected {
		return
	}
	for i := range list {
		task := &list[i]
		mark := time.Now()
		task.State = Failed
		task.Terminated = &mark
		task.Error("Error", "Hub is disconnected.")
		err = m.DB.Save(task).Error
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	return
}

// postpone Postpones a task as needed based on rules.
func (m *Manager) postpone(list []model.Task) (err error) {
	ruleSet := []Rule{
		&RuleIsolated{},
		&RuleUnique{},
		&RuleDeps{
			cluster: m.cluster,
		},
	}
	for i := range list {
		task := &list[i]
		if !(task.State == Ready || task.State == Postponed) {
			continue
		}
		ready := task
		for j := range list {
			other := &list[j]
			if ready.ID == other.ID {
				continue
			}
			if !(other.State == Running || other.State == Pending) {
				continue
			}
			for _, rule := range ruleSet {
				if rule.Match(ready, other) {
					Log.Info("Task postponed.", "id", ready.ID)
					err = m.DB.Save(task).Error
					if err != nil {
						err = liberr.Wrap(err)
						return
					}
				}
			}
		}
	}

	return
}

// adjustPriority escalate as needed.
// When adjusted, Pending tasks pods deleted and made Ready again.
func (m *Manager) adjustPriority(list []model.Task) (err error) {
	pE := Priority{cluster: m.cluster}
	escalated := pE.Escalate(list)
	for _, task := range escalated {
		Log.V(1).Info("Priority escalated.", "id", task.ID)
		if task.State != Pending {
			continue
		}
		rt := Task{task}
		err = rt.Delete(m.Client)
		if err != nil {
			return
		}
		task.State = Ready
		err = m.DB.Save(task).Error
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	return
}

// updateRunning tasks to reflect pod state.
func (m *Manager) updateRunning() {
	var err error
	defer func() {
		Log.Error(err, "")
	}()
	list := []model.Task{}
	db := m.DB.Order("priority DESC, id")
	result := db.Find(
		&list,
		"state IN ?",
		[]string{
			Pending,
			Running,
		})
	if result.Error != nil {
		err = liberr.Wrap(result.Error)
		return
	}
	err = m.canceled(list)
	if err != nil {
		return
	}
	for _, task := range list {
		if !(task.State == Running || task.State == Pending) {
			continue
		}
		running := task
		rt := Task{&running}
		pod, err := rt.Reflect(m.DB, m.cluster)
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
		err = m.DB.Save(&running).Error
		if err != nil {
			return
		}
		Log.V(1).Info("Task updated.", "id", running.ID)
	}
}

// The task has been canceled.
func (m *Manager) canceled(list []model.Task) (err error) {
	for i := range list {
		task := &list[i]
		if !task.Canceled {
			continue
		}
		rt := Task{task}
		err = rt.Cancel(m.Client)
		if err != nil {
			return
		}
		err = m.DB.Save(task).Error
		Log.Error(err, "")
		db := m.DB.Model(&model.TaskReport{})
		err = db.Delete("taskid", task.ID).Error
		if err != nil {
			err = liberr.Wrap(err)
			break
		}
	}
	return
}

// snapshotPod attaches a pod description and logs.
// Includes:
//   - pod YAML
//   - pod Events
//   - container Logs
func (m *Manager) snapshotPod(task *Task, pod *core.Pod) (err error) {
	var files []*model.File
	d, err := m.podYAML(pod)
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

// podYAML builds pod resource description.
func (m *Manager) podYAML(pod *core.Pod) (file *model.File, err error) {
	events, err := m.podEvent(pod)
	if err != nil {
		return
	}
	file = &model.File{Name: "pod.yaml"}
	err = m.DB.Create(file).Error
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
	type Pod struct {
		core.Pod `yaml:",inline"`
		Events   []Event `yaml:",omitempty"`
	}
	d := Pod{
		Pod:    *pod,
		Events: events,
	}
	b, _ := yaml.Marshal(d)
	_, _ = f.Write(b)
	return
}

// podEvent get pod events.
func (m *Manager) podEvent(pod *core.Pod) (events []Event, err error) {
	clientSet, err := k8s2.NewClientSet()
	if err != nil {
		return
	}
	options := meta.ListOptions{
		FieldSelector: "involvedObject.name=" + pod.Name,
		TypeMeta: meta.TypeMeta{
			Kind: "Pod",
		},
	}
	eventClient := clientSet.CoreV1().Events(Settings.Hub.Namespace)
	eventList, err := eventClient.List(context.TODO(), options)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for _, event := range eventList.Items {
		duration := event.LastTimestamp.Sub(event.FirstTimestamp.Time)
		events = append(
			events,
			Event{
				Type:     event.Type,
				Reason:   event.Reason,
				Age:      duration.String(),
				Reporter: event.ReportingController,
				Message:  event.Message,
			})
	}
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
	err = m.DB.Create(file).Error
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
func (r *Task) Run(db *gorm.DB, cluster Cluster) (err error) {
	client := cluster.Client
	mark := time.Now()
	defer func() {
		if err != nil {
			if !errors.Is(err, &QuotaExceeded{}) {
				r.Error("Error", "err.Error()")
				r.Terminated = &mark
				r.State = Failed
			}
		}
	}()
	err = r.selectAddon(db, cluster)
	if err != nil {
		return
	}
	priority, err := r.selectPriority(cluster)
	if err != nil {
		return
	}
	addon, found := cluster.addons[r.Addon]
	if !found {
		err = &AddonNotFound{Name: r.Addon}
		return
	}
	err = r.selectExtensions(db, cluster, addon)
	if err != nil {
		return
	}
	extensions, err := r.getExtensions(client)
	if err != nil {
		return
	}
	for _, extension := range extensions {
		if r.Addon != extension.Spec.Addon {
			err = &ExtensionNotValid{
				Name:  extension.Name,
				Addon: addon.Name,
			}
			return
		}
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
	pod := r.pod(
		priority,
		addon,
		extensions,
		cluster.tackle,
		&secret)
	err = client.Create(context.TODO(), &pod)
	if err != nil {
		quotaExceeded := &QuotaExceeded{}
		if quotaExceeded.Match(err) {
			err = quotaExceeded
		}
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
func (r *Task) Reflect(db *gorm.DB, cluster Cluster) (pod *core.Pod, err error) {
	pod, found := cluster.pods[path.Base(r.Pod)]
	if !found {
		err = r.Run(db, cluster)
		if errors.Is(err, &QuotaExceeded{}) {
			Log.V(1).Info(err.Error())
			err = nil
		}
		return
	}
	client := cluster.Client
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
	addonStatus := pod.Status.ContainerStatuses[0]
	if addonStatus.State.Terminated != nil {
		switch addonStatus.State.Terminated.ExitCode {
		case 0:
			r.podSucceeded(pod)
		default: // failed.
			r.podFailed(pod, client)
			return
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

// selectAddon select an addon when not specified.
func (r *Task) selectAddon(db *gorm.DB, cluster Cluster) (err error) {
	if r.Addon != "" {
		return
	}
	kind, found := cluster.tasks[r.Kind]
	if !found {
		return
	}
	selected := ""
	addons := kind.Spec.Addon
	for i := range addons {
		var selector Selector
		var matched []string
		resolver := &AddonResolver{
			task: kind.Name,
		}
		err = resolver.Load(cluster)
		if err != nil {
			return
		}
		selector, err = NewSelector(addons[i], resolver)
		if err != nil {
			return
		}
		matched, err = selector.Match(db, r.Task)
		if err != nil {
			return
		}
		if len(matched) > 0 {
			selected = matched[0]
			break
		}
	}
	if selected == "" {
		err = &AddonNotSelected{}
		return
	}
	r.Addon = selected
	return
}

// selectExtensions select extensions when not specified.
func (r *Task) selectExtensions(db *gorm.DB, cluster Cluster, addon *crd.Addon) (err error) {
	var extensions []string
	if r.Extensions != nil {
		_ = json.Unmarshal(r.Extensions, &extensions)
	}
	if len(extensions) > 0 {
		return
	}
	names := make(map[string]int)
	selectors := addon.Spec.Extension
	for i := range selectors {
		var selector Selector
		var matched []string
		resolver := &ExtensionResolver{
			addon: addon.Name,
		}
		err = resolver.Load(cluster)
		if err != nil {
			return
		}
		selector, err = NewSelector(selectors[i], resolver)
		if err != nil {
			return
		}
		matched, err = selector.Match(db, r.Task)
		if err != nil {
			return
		}
		for _, name := range matched {
			names[name] = 0
		}
	}
	extensions = make([]string, 0)
	for name := range names {
		extensions = append(
			extensions,
			name)
	}
	r.Extensions, _ = json.Marshal(extensions)
	return
}

// getExtensions by name.
func (r *Task) getExtensions(client k8s.Client) (extensions []crd.Extension, err error) {
	var names []string
	_ = json.Unmarshal(r.Extensions, &names)
	for _, name := range names {
		extension := crd.Extension{}
		err = client.Get(
			context.TODO(),
			k8s.ObjectKey{
				Namespace: Settings.Hub.Namespace,
				Name:      name,
			},
			&extension)
		if err != nil {
			if k8serr.IsNotFound(err) {
				err = &ExtensionNotFound{name}
			} else {
				err = liberr.Wrap(err)
			}
			return
		}
		extensions = append(
			extensions,
			extension)
	}
	return
}

// pod build the pod.
func (r *Task) pod(
	priority string,
	addon *crd.Addon,
	extensions []crd.Extension,
	owner *crd.Tackle,
	secret *core.Secret) (pod core.Pod) {
	//
	pod = core.Pod{
		Spec: r.specification(addon, extensions, secret),
		ObjectMeta: meta.ObjectMeta{
			Namespace:    Settings.Hub.Namespace,
			GenerateName: r.k8sName(),
			Labels:       r.labels(),
		},
	}
	pod.Spec.PriorityClassName = priority
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
func (r *Task) specification(
	addon *crd.Addon,
	extensions []crd.Extension,
	secret *core.Secret) (specification core.PodSpec) {
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
	init, plain := r.containers(addon, extensions, secret)
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
	addon *crd.Addon,
	extensions []crd.Extension,
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
	plain = append(plain, addon.Spec.Container)
	plain[0].Name = "addon"
	for i := range extensions {
		extension := &extensions[i]
		container := extension.Spec.Container
		container.Name = extension.Name
		plain = append(
			plain,
			container)
	}
	injector := Injector{}
	for i := range plain {
		container := &plain[i]
		injector.Inject(container)
		r.propagateEnv(&plain[0], container)
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
				Name:  settings.EnvSharedPath,
				Value: Settings.Shared.Path,
			},
			core.EnvVar{
				Name:  settings.EnvCachePath,
				Value: Settings.Cache.Path,
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
	}
	return
}

// propagateEnv copies extension container Env.* to the addon container.
// Prefixed with EXTENSION_<name>.
func (r *Task) propagateEnv(addon, extension *core.Container) {
	for _, env := range extension.Env {
		addon.Env = append(
			addon.Env,
			core.EnvVar{
				Name:  ExtEnv(extension.Name, env.Name),
				Value: env.Value,
			})
	}
}

// secret builds the pod secret.
func (r *Task) secret() (secret core.Secret) {
	user := "addon:" + r.Addon
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

// selectPriority sets the pod priority class.
func (r *Task) selectPriority(cluster Cluster) (name string, err error) {
	if r.Priority > 0 {
		p, found := cluster.priority.values[r.Priority]
		if !found {
			err = &PriorityNotFound{Value: r.Priority}
		} else {
			name = p.Name
		}
		return
	}
	kind, found := cluster.tasks[r.Kind]
	if found {
		name = kind.Spec.Priority
		p, found := cluster.priority.names[name]
		if !found {
			err = &PriorityNotFound{Name: name}
			return
		}
		r.Priority = int(p.Value)
	}
	return
}

// Event represents a pod event.
type Event struct {
	Type     string
	Reason   string
	Age      string
	Reporter string
	Message  string
}

// Priority escalator.
type Priority struct {
	cluster Cluster
}

// Escalate tasks as needed.
func (p *Priority) Escalate(ready []model.Task) (escalated []*model.Task) {
	_, escalated = p.escalate(ready)
	escalated = p.unique(escalated)
	return
}

// escalate tasks.
func (p *Priority) escalate(ready []model.Task) (pushed, escalated []*model.Task) {
	for i := range ready {
		task := &ready[i]
		if task.State != Ready {
			continue
		}
		kind, found := p.cluster.tasks[task.Kind]
		if !found {
			continue
		}
		pushed = append(pushed, task)
		for _, d := range kind.Spec.Dependencies {
			next := ready[i+1:]
			for r := range next {
				nt := &next[r]
				if nt.Kind == d &&
					nt.ApplicationID == task.ApplicationID {
					innerPushed, innerEscalated := p.escalate(next[r:])
					pushed = append(
						pushed,
						innerPushed...)
					escalated = append(
						escalated,
						innerEscalated...)
				}
			}
		}
		p0 := pushed[0].Priority
		for p := range pushed[1:] {
			pP := pushed[p].Priority
			if pP < p0 {
				pushed[p].Priority = p0
				escalated = append(
					escalated,
					pushed[p])
			}
		}
	}
	return
}

// unique returns a unique list of tasks.
func (p *Priority) unique(in []*model.Task) (out []*model.Task) {
	mp := make(map[uint]*model.Task)
	for _, ptr := range in {
		mp[ptr.ID] = ptr
	}
	for _, ptr := range mp {
		out = append(out, ptr)
	}
	return
}

type Cluster struct {
	k8s.Client
	tackle     *crd.Tackle
	addons     map[string]*crd.Addon
	extensions map[string]*crd.Extension
	tasks      map[string]*crd.Task
	priority   struct {
		names  map[string]*sched.PriorityClass
		values map[int]*sched.PriorityClass
	}
	pods map[string]*core.Pod
}

func (k *Cluster) Refresh() (err error) {
	err = k.getTackle()
	if err != nil {
		return
	}
	err = k.getAddons()
	if err != nil {
		return
	}
	err = k.getExtensions()
	if err != nil {
		return
	}
	err = k.getTasks()
	if err != nil {
		return
	}
	err = k.getPriorities()
	if err != nil {
		return
	}
	err = k.getPods()
	if err != nil {
		return
	}
	return
}

// getTackle
func (k *Cluster) getTackle() (err error) {
	options := &k8s.ListOptions{Namespace: Settings.Namespace}
	list := crd.TackleList{}
	err = k.List(
		context.TODO(),
		&list,
		options)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for i := range list.Items {
		r := &list.Items[i]
		k.tackle = r
		return
	}
	err = liberr.New("Tackle CR not found.")
	return
}

// getAddons
func (k *Cluster) getAddons() (err error) {
	k.addons = make(map[string]*crd.Addon)
	options := &k8s.ListOptions{Namespace: Settings.Namespace}
	list := crd.AddonList{}
	err = k.List(
		context.TODO(),
		&list,
		options)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for i := range list.Items {
		r := &list.Items[i]
		k.addons[r.Name] = r
	}
	return
}

// getExtensions
func (k *Cluster) getExtensions() (err error) {
	k.extensions = make(map[string]*crd.Extension)
	options := &k8s.ListOptions{Namespace: Settings.Namespace}
	list := crd.ExtensionList{}
	err = k.List(
		context.TODO(),
		&list,
		options)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for i := range list.Items {
		r := &list.Items[i]
		k.extensions[r.Name] = r
	}
	return
}

// getTasks kinds.
func (k *Cluster) getTasks() (err error) {
	k.tasks = make(map[string]*crd.Task)
	options := &k8s.ListOptions{Namespace: Settings.Namespace}
	list := crd.TaskList{}
	err = k.List(
		context.TODO(),
		&list,
		options)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for i := range list.Items {
		r := &list.Items[i]
		k.tasks[r.Name] = r
	}
	return
}

// getPriorities classes.
func (k *Cluster) getPriorities() (err error) {
	k.priority.names = make(map[string]*sched.PriorityClass)
	k.priority.values = make(map[int]*sched.PriorityClass)
	list := sched.PriorityClassList{}
	err = k.List(context.TODO(), &list)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for i := range list.Items {
		r := &list.Items[i]
		k.priority.names[r.Name] = r
		k.priority.values[int(r.Value)] = r
	}
	return
}

// getPods
func (k *Cluster) getPods() (err error) {
	k.pods = make(map[string]*core.Pod)
	options := &k8s.ListOptions{Namespace: Settings.Namespace}
	list := core.PodList{}
	err = k.List(
		context.TODO(),
		&list,
		options)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	for i := range list.Items {
		r := &list.Items[i]
		k.pods[r.Name] = r
	}
	return
}

// ExtEnv returns an environment variable named namespaced to an extension.
// Format: _EXT_<extension_<var>.
func ExtEnv(extension string, envar string) (s string) {
	s = strings.Join(
		[]string{
			"_EXT",
			strings.ToUpper(extension),
			envar,
		},
		"_")
	return
}
