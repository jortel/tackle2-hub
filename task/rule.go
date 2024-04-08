package task

import (
	"strings"

	crd "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	"github.com/konveyor/tackle2-hub/model"
)

// Rule defines postpone rules.
type Rule interface {
	Match(candidate, other *model.Task) bool
}

// RuleUnique running tasks must be unique by:
//   - application
//   - addon.
type RuleUnique struct {
}

// Match determines the match.
func (r *RuleUnique) Match(candidate, other *model.Task) (matched bool) {
	if candidate.ApplicationID == nil || other.ApplicationID == nil {
		return
	}
	if *candidate.ApplicationID != *other.ApplicationID {
		return
	}
	if candidate.Addon != other.Addon {
		return
	}
	matched = true
	Log.Info(
		"Rule:Unique matched.",
		"candidate",
		candidate.ID,
		"by",
		other.ID)

	return
}

// RuleDeps - Task kind dependencies.
type RuleDeps struct {
	kinds map[string]crd.Task
}

// Match determines the match.
func (r *RuleDeps) Match(candidate, other *model.Task) (matched bool) {
	if candidate.Kind == "" || other.Kind == "" {
		return
	}
	if *candidate.ApplicationID != *other.ApplicationID {
		return
	}
	def, found := r.kinds[candidate.Kind]
	if !found {
		return
	}
	matched = def.HasDep(other.Kind)
	Log.Info(
		"Rule:dep matched.",
		"candidate",
		candidate.ID,
		"by",
		other.ID)
	return
}

// RuleIsolated policy.
type RuleIsolated struct {
}

// Match determines the match.
func (r *RuleIsolated) Match(candidate, other *model.Task) (matched bool) {
	matched = hasPolicy(candidate, Isolated) || hasPolicy(other, Isolated)
	if matched {
		Log.Info(
			"Rule:Isolated matched.",
			"candidate",
			candidate.ID,
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
