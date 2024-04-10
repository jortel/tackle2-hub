package task

import (
	"errors"
	"fmt"
	"strconv"
)

// KindNotFound used to report profile referenced
// by a task but cannot be found.
type KindNotFound struct {
	Name string
}

func (e *KindNotFound) Error() (s string) {
	return fmt.Sprintf("Task (kind): '%s' not-found.", e.Name)
}

func (e *KindNotFound) Is(err error) (matched bool) {
	var inst *KindNotFound
	matched = errors.As(err, &inst)
	return
}

// AddonNotFound used to report addon referenced
// by a task but cannot be found.
type AddonNotFound struct {
	Name string
}

func (e *AddonNotFound) Error() (s string) {
	return fmt.Sprintf("Addon: '%s' not-found.", e.Name)
}

func (e *AddonNotFound) Is(err error) (matched bool) {
	var inst *AddonNotFound
	matched = errors.As(err, &inst)
	return
}

// AddonNotSelected report that an addon has not been selected.
type AddonNotSelected struct {
}

func (e *AddonNotSelected) Error() (s string) {
	return fmt.Sprintf("Addon not selected.")
}

func (e *AddonNotSelected) Is(err error) (matched bool) {
	var inst *AddonNotSelected
	matched = errors.As(err, &inst)
	return
}

// ExtensionNotFound used to report extension referenced
// by a task but cannot be found.
type ExtensionNotFound struct {
	Name string
}

func (e *ExtensionNotFound) Error() (s string) {
	return fmt.Sprintf("Extension: '%s' not-found.", e.Name)
}

func (e *ExtensionNotFound) Is(err error) (matched bool) {
	var inst *ExtensionNotFound
	matched = errors.As(err, &inst)
	return
}

// ExtensionNotValid used to report extension referenced
// by a task not valid with addon.
type ExtensionNotValid struct {
	Name  string
	Addon string
}

func (e *ExtensionNotValid) Error() (s string) {
	return fmt.Sprintf(
		"Extension: '%s' not-valid with addon '%s'.",
		e.Name,
		e.Addon)
}

func (e *ExtensionNotValid) Is(err error) (matched bool) {
	var inst *ExtensionNotValid
	matched = errors.As(err, &inst)
	return
}

// SelectorNotSupported reports unknown selector.
type SelectorNotSupported struct {
	Kind string
}

func (e *SelectorNotSupported) Error() (s string) {
	return fmt.Sprintf("Selector: '%s' not supported.", e.Kind)
}

func (e *SelectorNotSupported) Is(err error) (matched bool) {
	var inst *SelectorNotSupported
	matched = errors.As(err, &inst)
	return
}

// NotResolved report name/capability not resolved.
type NotResolved struct {
	Kind string
	Name string
}

func (e *NotResolved) Error() (s string) {
	return fmt.Sprintf("%s: '%s' not-resolved.", e.Kind, e.Name)
}

func (e *NotResolved) Is(err error) (matched bool) {
	var inst *NotResolved
	matched = errors.As(err, &inst)
	return
}

// PriorityNotFound report priority class not found.
type PriorityNotFound struct {
	Name  string
	Value int
}

func (e *PriorityNotFound) Error() (s string) {
	var d string
	if e.Name != "" {
		d = fmt.Sprintf("\"%s\"", e.Name)
	} else {
		d = strconv.Itoa(e.Value)
	}
	s = fmt.Sprintf("Priority %s not-found.", d)
	return
}

func (e *PriorityNotFound) Is(err error) (matched bool) {
	var inst *PriorityNotFound
	matched = errors.As(err, &inst)
	return
}
