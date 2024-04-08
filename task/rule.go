package task

import (
	"strings"

	crd "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	"github.com/konveyor/tackle2-hub/model"
)

// Rule defines postpone rules.
type Rule interface {
	Match(ready, other *model.Task) bool
}

// RuleUnique running tasks must be unique by:
//   - application
//   - addon.
type RuleUnique struct {
}

// Match determines the match.
func (r *RuleUnique) Match(ready, other *model.Task) (matched bool) {
	if ready.ApplicationID == nil || other.ApplicationID == nil {
		return
	}
	if *ready.ApplicationID != *other.ApplicationID {
		return
	}
	if ready.Addon != other.Addon {
		return
	}
	matched = true
	Log.Info(
		"Rule:Unique matched.",
		"ready",
		ready.ID,
		"by",
		other.ID)

	return
}

// RuleDeps - Task kind dependencies.
type RuleDeps struct {
	kinds map[string]crd.Task
}

// Match determines the match.
func (r *RuleDeps) Match(ready, other *model.Task) (matched bool) {
	if ready.Kind == "" || other.Kind == "" {
		return
	}
	if *ready.ApplicationID != *other.ApplicationID {
		return
	}
	def, found := r.kinds[ready.Kind]
	if !found {
		return
	}
	matched = def.HasDep(other.Kind)
	Log.Info(
		"Rule:dep matched.",
		"ready",
		ready.ID,
		"by",
		other.ID)
	return
}

// RuleIsolated policy.
type RuleIsolated struct {
}

// Match determines the match.
func (r *RuleIsolated) Match(ready, other *model.Task) (matched bool) {
	matched = hasPolicy(ready, Isolated) || hasPolicy(other, Isolated)
	if matched {
		Log.Info(
			"Rule:Isolated matched.",
			"ready",
			ready.ID,
			"by",
			other.ID)
	}

	return
}

// Returns true if the task policy includes the specified rule.
func hasPolicy(task *model.Task, name string) (matched bool) {
	for _, p := range strings.Split(task.Policy, ";") {
		p = strings.TrimSpace(p)
		p = strings.ToLower(p)
		if p == name {
			matched = true
			break
		}
	}

	return
}
