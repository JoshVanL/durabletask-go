package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/dapr/durabletask-go/api"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/backend/runtimestate"
	"github.com/dapr/kit/ptr"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrTaskHubExists         = errors.New("task hub already exists")
	ErrTaskHubNotFound       = errors.New("task hub not found")
	ErrNotInitialized        = errors.New("backend not initialized")
	ErrWorkItemLockLost      = errors.New("lock on work-item was lost")
	ErrBackendAlreadyStarted = errors.New("backend is already started")
)

type Backend interface {
	// CreateTaskHub creates a new task hub for the current backend. Task hub creation must be idempotent.
	//
	// If the task hub for this backend already exists, an error of type [ErrTaskHubExists] is returned.
	CreateTaskHub(context.Context) error

	// DeleteTaskHub deletes an existing task hub configured for the current backend. It's up to the backend
	// implementation to determine how the task hub data is deleted.
	//
	// If the task hub for this backend doesn't exist, an error of type [ErrTaskHubNotFound] is returned.
	DeleteTaskHub(context.Context) error

	// Start starts any background processing done by this backend.
	Start(context.Context) error

	// Stop stops any background processing done by this backend.
	Stop(context.Context) error

	// CreateWorkflowInstance creates a new orchestration instance with a history event that
	// wraps a ExecutionStarted event.
	CreateWorkflowInstance(context.Context, *protos.HistoryEvent) error

	// RerunWorkflowFromEvent reruns a workflow from a specific event ID of some
	// source instance ID. If not given, a random new instance ID will be
	// generated and returned. Can optionally give a new input to the target
	// event ID to rerun from.
	RerunWorkflowFromEvent(ctx context.Context, req *protos.RerunWorkflowFromEventRequest) (string, error)

	// AddNewEvent adds a new orchestration event to the specified orchestration instance.
	AddNewWorkflowEvent(context.Context, string, *protos.HistoryEvent) error

	// NextWorkflowWorkItem blocks and returns the next orchestration work
	// item from the task hub. Should only return an error when shutting down.
	NextWorkflowWorkItem(context.Context) (*WorkflowWorkItem, error)

	// GetWorkflowRuntimeState gets the runtime state of an orchestration instance.
	GetWorkflowRuntimeState(context.Context, *WorkflowWorkItem) (*protos.WorkflowRuntimeState, error)

	// WatchWorkflowRuntimeStatus is a streaming API to watch for changes to
	// the OrchestrtionMetadata, receiving events as and when the state changes.
	// When the given condition is true, returns.
	// Used over polling the metadata.
	WatchWorkflowRuntimeStatus(ctx context.Context, id string, condition func(*protos.WorkflowMetadata) bool) error

	// GetWorkflowMetadata gets the metadata associated with the given orchestration instance ID.
	//
	// Returns [api.ErrInstanceNotFound] if the orchestration instance doesn't exist.
	GetWorkflowMetadata(context.Context, string) (*protos.WorkflowMetadata, error)

	// CompleteWorkflowWorkItem completes a work item by saving the updated runtime state to durable storage.
	//
	// Returns [ErrWorkItemLockLost] if the work-item couldn't be completed due to a lock-lost conflict (e.g., split-brain).
	CompleteWorkflowWorkItem(context.Context, *WorkflowWorkItem) error

	// AbandonWorkflowWorkItem undoes any state changes and returns the work item to the work item queue.
	//
	// This is called if an internal failure happens in the processing of an orchestration work item. It is
	// not called if the orchestration work item is processed successfully (note that an orchestration that
	// completes with a failure is still considered a successfully processed work item).
	AbandonWorkflowWorkItem(context.Context, *WorkflowWorkItem) error

	// NextActivityWorkItem blocks and returns the next activity work item from
	// the task hub. Should only return an error when shutting down.
	NextActivityWorkItem(context.Context) (*ActivityWorkItem, error)

	// CompleteActivityWorkItem sends a message to the parent orchestration indicating activity completion.
	//
	// Returns [ErrWorkItemLockLost] if the work-item couldn't be completed due to a lock-lost conflict (e.g., split-brain).
	CompleteActivityWorkItem(context.Context, *ActivityWorkItem) error

	// AbandonActivityWorkItem returns the work-item back to the queue without committing any other chances.
	//
	// This is called when an internal failure occurs during activity work-item processing.
	AbandonActivityWorkItem(context.Context, *ActivityWorkItem) error

	// PurgeWorkflowState deletes all saved state for the specified orchestration instance.
	//
	// [api.ErrInstanceNotFound] is returned if the specified orchestration instance doesn't exist.
	// [api.ErrNotCompleted] is returned if the specified orchestration instance is still running.
	PurgeWorkflowState(ctx context.Context, id string, force bool) error

	// CompleteWorkflowTask completes the orchestrator task by saving the updated runtime state to durable storage.
	CompleteWorkflowTask(context.Context, *protos.WorkflowResponse) error

	// CancelWorkflowTask cancels the orchestrator task so instances of WaitForWorkflowCompletion will return an error.
	CancelWorkflowTask(context.Context, string) error

	// WaitForWorkflowCompletion blocks until the orchestrator completes and returns the final response.
	//
	// [api.ErrTaskCancelled] is returned if the task was cancelled.
	WaitForWorkflowCompletion(*protos.WorkflowRequest) func(context.Context) (*protos.WorkflowResponse, error)

	// CompleteActivityTask completes the activity task by saving the updated runtime state to durable storage.
	CompleteActivityTask(context.Context, *protos.ActivityResponse) error

	// CancelActivityTask cancels the activity task so instances of WaitForActivityCompletion will return an error.
	CancelActivityTask(context.Context, string, int32) error

	// WaitForActivityCompletion blocks until the activity completes and returns the final response.
	//
	// [api.ErrTaskCancelled] is returned if the task was cancelled.
	WaitForActivityCompletion(*protos.ActivityRequest) func(context.Context) (*protos.ActivityResponse, error)

	// ListInstanceIDs lists orchestration instance IDs based on the provided
	// query parameters.
	ListInstanceIDs(ctx context.Context, req *protos.ListInstanceIDsRequest) (*protos.ListInstanceIDsResponse, error)

	// GetInstanceHistory returns the full current history of a workflow instance.
	GetInstanceHistory(ctx context.Context, req *protos.GetInstanceHistoryRequest) (*protos.GetInstanceHistoryResponse, error)
}

// MarshalHistoryEvent serializes the [HistoryEvent] into a protobuf byte array.
func MarshalHistoryEvent(e *protos.HistoryEvent) ([]byte, error) {
	if bytes, err := proto.Marshal(e); err != nil {
		return nil, fmt.Errorf("failed to marshal history event: %w", err)
	} else {
		return bytes, nil
	}
}

// UnmarshalHistoryEvent deserializes a [HistoryEvent] from a protobuf byte array.
func UnmarshalHistoryEvent(bytes []byte) (*protos.HistoryEvent, error) {
	e := &protos.HistoryEvent{}
	if err := proto.Unmarshal(bytes, e); err != nil {
		return nil, fmt.Errorf("unreadable history event payload: %w", err)
	}
	return e, nil
}

// purgeWorkflowState purges the orchestration state, including sub-orchestrations if [recursive] is true.
// Returns (deletedInstanceCount, error), where deletedInstanceCount is the number of instances deleted.
func purgeWorkflowState(ctx context.Context, be Backend, iid string, recursive bool, force bool) (int, error) {
	deletedInstanceCount := 0
	if recursive {
		owi := &WorkflowWorkItem{
			InstanceID: iid,
		}
		state, err := be.GetWorkflowRuntimeState(ctx, owi)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch orchestration state: %w", err)
		}
		if len(state.NewEvents)+len(state.OldEvents) == 0 {
			// If there are no events, the orchestration instance doesn't exist
			return 0, api.ErrInstanceNotFound
		}
		if !runtimestate.IsCompleted(state) {
			// Workflow must be completed before purging its state
			return 0, api.ErrNotCompleted
		}
		subWorkflowInstances := getSubWorkflowInstances(state.OldEvents, state.NewEvents)
		for _, subWorkflowInstance := range subWorkflowInstances {
			// Recursively purge sub-orchestrations
			count, err := purgeWorkflowState(ctx, be, subWorkflowInstance, recursive, force)
			// `count` sub-orchestrations have been successfully purged (even in case of error)
			deletedInstanceCount += count
			if err != nil {
				return deletedInstanceCount, fmt.Errorf("failed to purge sub-orchestration: %w", err)
			}
		}
	}
	// Purging root orchestration
	if err := be.PurgeWorkflowState(ctx, iid, force); err != nil {
		return deletedInstanceCount, err
	}
	return deletedInstanceCount + 1, nil
}

func terminateChildWorkflowInstances(ctx context.Context, be Backend, iid string, state *protos.WorkflowRuntimeState, et *protos.ExecutionTerminatedEvent) error {
	if !et.Recurse {
		return nil
	}
	subWorkflowInstances := getSubWorkflowInstances(state.OldEvents, state.NewEvents)
	for _, subWorkflowInstance := range subWorkflowInstances {
		e := &protos.HistoryEvent{
			EventID:   ptr.Of(int32(-1)),
			Timestamp: timestamppb.Now(),
			EventType: &protos.HistoryEvent_ExecutionTerminated{
				ExecutionTerminated: &protos.ExecutionTerminatedEvent{
					Input:   et.Input,
					Recurse: et.Recurse,
				},
			},
		}
		// Adding terminate event to sub-orchestration instance
		if err := be.AddNewWorkflowEvent(ctx, subWorkflowInstance, e); err != nil {
			return fmt.Errorf("failed to submit termination request to sub-orchestration: %w", err)
		}
	}
	return nil
}

// getSubWorkflowInstances returns the instance IDs of all sub-orchestrations in the specified events.
func getSubWorkflowInstances(oldEvents []*protos.HistoryEvent, newEvents []*protos.HistoryEvent) []string {
	subWorkflowInstancesMap := make(map[string]struct{}, len(oldEvents)+len(newEvents))
	for _, e := range oldEvents {
		if created := e.GetChildWorkflowInstanceCreated(); created != nil {
			subWorkflowInstancesMap[created.InstanceID] = struct{}{}
		}
	}
	for _, e := range newEvents {
		if created := e.GetChildWorkflowInstanceCreated(); created != nil {
			subWorkflowInstancesMap[string(created.InstanceID)] = struct{}{}
		}
	}
	subWorkflowInstances := make([]string, 0, len(subWorkflowInstancesMap))
	for orch := range subWorkflowInstancesMap {
		subWorkflowInstances = append(subWorkflowInstances, orch)
	}
	return subWorkflowInstances
}
