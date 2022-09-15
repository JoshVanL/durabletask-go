package backend

import (
	"context"
	"errors"

	"github.com/cgillum/durabletask-go/api"
	"github.com/cgillum/durabletask-go/internal/protos"
)

var ErrTaskHubExists = errors.New("task hub already exists")
var ErrTaskHubNotFound = errors.New("task hub not found")
var ErrNotInitialized = errors.New("backend not initialized")
var ErrWorkItemLockLost = errors.New("lock on work-item was lost")

type Backend interface {
	// CreateTaskHub creates a new task hub for the current backend. It's up to the backend implementation
	// and its configuration to determine how it gets created.
	//
	// If the task hub for this backend already exists, an error of type ErrTaskHubExists is returned.
	CreateTaskHub(context.Context) error

	// DeleteTaskHub deletes an existing task hub configured for the current backend. It's up to the backend
	// implementation to determine how the task hub data is deleted.
	//
	// If the task hub for this backend doesn't exist, an error of type ErrTaskHubNotFound is returned.
	DeleteTaskHub(context.Context) error

	// Start starts any background processing done by this backend.
	Start(context.Context) error

	// Stop stops any background processing done by this backend.
	Stop(context.Context) error

	// CreateOrchestrationInstance creates a new orchestration instance with a history event that
	// wraps a ExecutionStarted event.
	CreateOrchestrationInstance(context.Context, *protos.HistoryEvent) error

	// AddNewEvent adds a new orchestration event to the specified orchestration instance.
	AddNewOrchestrationEvent(context.Context, api.InstanceID, *protos.HistoryEvent) error

	// GetOrchestrationWorkItem gets a pending work item from the task hub or returns ErrNoOrchWorkItems
	// if there are no pending work items.
	GetOrchestrationWorkItem(context.Context) (*OrchestrationWorkItem, error)

	// GetOrchestrationRuntimeState gets the runtime state of an orchestration instance.
	GetOrchestrationRuntimeState(context.Context, *OrchestrationWorkItem) (*OrchestrationRuntimeState, error)

	// GetOrchestrationMetadata gets the metadata associated with the given orchestration instance ID.
	GetOrchestrationMetadata(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error)

	// CompleteOrchestrationWorkItem completes a work item by saving the updated runtime state to durable storage.
	//
	// Returns ErrWorkItemLockLost if the work-item couldn't be completed due to a lock-lost conflict (e.g., split-brain).
	CompleteOrchestrationWorkItem(context.Context, *OrchestrationWorkItem) error

	// AbandonOrchestrationWorkItem undoes any state changes and returns the work item to the work item queue.
	//
	// This is called if an internal failure happens in the processing of an orchestration work item. It is
	// not called if the orchestration work item is processed successfully (note that an orchestration that
	// completes with a failure is still considered a successfully processed work item).
	AbandonOrchestrationWorkItem(context.Context, *OrchestrationWorkItem) error

	// GetActivityWorkItem gets a pending activity work item from the task hub or returns ErrNoWorkItems
	// if there are no pending activity work items.
	GetActivityWorkItem(context.Context) (*ActivityWorkItem, error)

	// CompleteActivityWorkItem sends a message to the parent orchestration indicating activity completion.
	//
	// Returns ErrWorkItemLockLost if the work-item couldn't be completed due to a lock-lost conflict (e.g., split-brain).
	CompleteActivityWorkItem(context.Context, *ActivityWorkItem) error

	// AbandonActivityWorkItem returns the work-item back to the queue without committing any other chances.
	//
	// This is called when an internal failure occurs during activity work-item processing.
	AbandonActivityWorkItem(context.Context, *ActivityWorkItem) error
}
