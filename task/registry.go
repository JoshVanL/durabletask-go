package task

import (
	"fmt"

	"github.com/dapr/durabletask-go/api/helpers"
)

// TaskRegistry contains maps of names to corresponding orchestrator and activity functions.
type TaskRegistry struct {
	orchestrators            map[string]Workflow
	versionedWorkflows       map[string]map[string]Workflow
	latestVersionedWorkflows map[string]string
	activities               map[string]Activity
}

// NewTaskRegistry returns a new [TaskRegistry] struct.
func NewTaskRegistry() *TaskRegistry {
	r := &TaskRegistry{
		orchestrators:            make(map[string]Workflow),
		activities:               make(map[string]Activity),
		versionedWorkflows:       make(map[string]map[string]Workflow),
		latestVersionedWorkflows: make(map[string]string),
	}
	return r
}

// AddWorkflow adds an orchestrator function to the registry. The name of the orchestrator
// function is determined using reflection.
func (r *TaskRegistry) AddWorkflow(o Workflow) error {
	name := helpers.GetTaskFunctionName(o)
	return r.AddWorkflowN(name, o)
}

// AddWorkflowN adds an orchestrator function to the registry with a specified name.
func (r *TaskRegistry) AddWorkflowN(name string, o Workflow) error {
	if _, ok := r.orchestrators[name]; ok {
		return fmt.Errorf("orchestrator named '%s' is already registered", name)
	}
	r.orchestrators[name] = o
	return nil
}

// AddVersionedWorkflow adds a versioned orchestrator function to the registry with a specified name.
func (r *TaskRegistry) AddVersionedWorkflow(canonicalName string, isLatest bool, o Workflow) error {
	name := helpers.GetTaskFunctionName(o)
	return r.AddVersionedWorkflowN(canonicalName, name, isLatest, o)
}

// AddVersionedWorkflowN adds a versioned orchestrator function to the registry with a specified name.
func (r *TaskRegistry) AddVersionedWorkflowN(canonicalName string, name string, isLatest bool, o Workflow) error {
	if _, ok := r.versionedWorkflows[canonicalName]; !ok {
		r.versionedWorkflows[canonicalName] = make(map[string]Workflow)
	}
	if _, ok := r.versionedWorkflows[canonicalName][name]; ok {
		return fmt.Errorf("versioned orchestrator named '%s' is already registered", name)
	}
	r.versionedWorkflows[canonicalName][name] = o
	if isLatest {
		r.latestVersionedWorkflows[canonicalName] = name
	}
	return nil
}

// AddActivity adds an activity function to the registry. The name of the activity
// function is determined using reflection.
func (r *TaskRegistry) AddActivity(a Activity) error {
	name := helpers.GetTaskFunctionName(a)
	return r.AddActivityN(name, a)
}

// AddActivityN adds an activity function to the registry with a specified name.
func (r *TaskRegistry) AddActivityN(name string, a Activity) error {
	if _, ok := r.activities[name]; ok {
		return fmt.Errorf("activity named '%s' is already registered", name)
	}
	r.activities[name] = a
	return nil
}
