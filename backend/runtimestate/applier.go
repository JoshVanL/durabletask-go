package runtimestate

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/kit/ptr"
)

type Applier struct {
	appID string
}

func NewApplier(appID string) *Applier {
	return &Applier{
		appID: appID,
	}
}

// Actions takes a set of actions and updates its internal state, including populating the outbox.
func (a *Applier) Actions(s *protos.WorkflowRuntimeState, customStatus *wrapperspb.StringValue, actions []*protos.WorkflowAction, currentTraceContext *protos.TraceContext) (bool, error) {
	s.CustomStatus = customStatus
	s.Stalled = nil

	for _, action := range actions {
		if action.Router == nil {
			action.Router = &protos.TaskRouter{
				SourceAppID: a.appID,
			}
		} else {
			action.Router.SourceAppID = a.appID
		}

		if completedAction := action.GetCompleteWorkflow(); completedAction != nil {
			if completedAction.WorkflowStatus == protos.WorkflowStatus_WORKFLOW_STATUS_CONTINUED_AS_NEW {
				newState := NewWorkflowRuntimeState(s.InstanceID, customStatus, []*protos.HistoryEvent{})
				newState.ContinuedAsNew = true
				_ = AddEvent(newState, &protos.HistoryEvent{
					EventID:   ptr.Of(int32(-1)),
					Timestamp: timestamppb.Now(),
					EventType: &protos.HistoryEvent_WorkflowStarted{
						WorkflowStarted: &protos.WorkflowStartedEvent{},
					},
					Router: action.Router,
				})

				// Duplicate the start event info, updating just the input
				_ = AddEvent(newState,
					&protos.HistoryEvent{
						EventID:   ptr.Of(int32(-1)),
						Timestamp: timestamppb.New(time.Now()),
						EventType: &protos.HistoryEvent_ExecutionStarted{
							ExecutionStarted: &protos.ExecutionStartedEvent{
								Name:           s.StartEvent.Name,
								ParentInstance: s.StartEvent.ParentInstance,
								Input:          completedAction.Result,
								WorkflowInstance: &protos.WorkflowInstance{
									InstanceID:  s.InstanceID,
									ExecutionId: wrapperspb.String(uuid.New().String()),
								},
								ParentTraceContext: s.StartEvent.ParentTraceContext,
							},
						},
						Router: action.Router,
					},
				)

				// Unprocessed "carryover" events
				for _, e := range completedAction.CarryoverEvents {
					_ = AddEvent(newState, e)
				}

				// Overwrite the current state object with a new one
				*s = *newState

				// ignore all remaining actions
				return true, nil
			} else {
				AddEvent(s, &protos.HistoryEvent{
					EventID:   ptr.Of(action.Id),
					Timestamp: timestamppb.Now(),
					EventType: &protos.HistoryEvent_ExecutionCompleted{
						ExecutionCompleted: &protos.ExecutionCompletedEvent{
							WorkflowStatus: completedAction.WorkflowStatus,
							Result:         completedAction.Result,
							FailureDetails: completedAction.FailureDetails,
						},
					},
					Router: action.Router,
				})
				if parentInstance := s.StartEvent.GetParentInstance(); parentInstance != nil {
					var completionRouter *protos.TaskRouter

					if parentInstance.AppID != nil {
						completionRouter = &protos.TaskRouter{
							SourceAppID: action.Router.GetSourceAppID(),
							TargetAppID: ptr.Of(parentInstance.GetAppID()),
						}
					} else {
						completionRouter = action.Router
					}

					msg := &protos.WorkflowRuntimeStateMessage{
						HistoryEvent: &protos.HistoryEvent{
							EventID:   ptr.Of(int32(-1)),
							Timestamp: timestamppb.Now(),
							Router:    completionRouter,
						},
						TargetInstanceID: s.StartEvent.GetParentInstance().WorkflowInstance.InstanceID,
					}
					if completedAction.WorkflowStatus == protos.WorkflowStatus_WORKFLOW_STATUS_COMPLETED {
						msg.HistoryEvent.EventType = &protos.HistoryEvent_ChildWorkflowInstanceCompleted{
							ChildWorkflowInstanceCompleted: &protos.ChildWorkflowInstanceCompletedEvent{
								TaskScheduledID: s.StartEvent.ParentInstance.TaskScheduledId,
								Result:          completedAction.Result,
							},
						}
					} else {
						// TODO: What is the expected result for termination?
						msg.HistoryEvent.EventType = &protos.HistoryEvent_ChildWorkflowInstanceFailed{
							ChildWorkflowInstanceFailed: &protos.ChildWorkflowInstanceFailedEvent{
								TaskScheduledID: s.StartEvent.ParentInstance.TaskScheduledId,
								FailureDetails:  completedAction.FailureDetails,
							},
						}
					}
					s.PendingMessages = append(s.PendingMessages, msg)
				}
			}
		} else if createtimer := action.GetCreateTimer(); createtimer != nil {
			_ = AddEvent(s, &protos.HistoryEvent{
				EventID:   ptr.Of(action.Id),
				Timestamp: timestamppb.New(time.Now()),
				EventType: &protos.HistoryEvent_TimerCreated{
					TimerCreated: &protos.TimerCreatedEvent{
						FireAt: createtimer.FireAt,
						Name:   createtimer.Name,
					},
				},
				Router: action.Router,
			})
			// TODO cant pass trace context
			s.PendingTimers = append(s.PendingTimers, &protos.HistoryEvent{
				EventID:   ptr.Of(int32(-1)),
				Timestamp: timestamppb.New(time.Now()),
				EventType: &protos.HistoryEvent_TimerFired{
					TimerFired: &protos.TimerFiredEvent{
						TimerID: action.Id,
						FireAt:  createtimer.FireAt,
					},
				},
			})
		} else if scheduleTask := action.GetScheduleTask(); scheduleTask != nil {
			scheduledEvent := &protos.HistoryEvent{
				EventID:   ptr.Of(action.Id),
				Timestamp: timestamppb.New(time.Now()),
				EventType: &protos.HistoryEvent_TaskScheduled{
					TaskScheduled: &protos.TaskScheduledEvent{
						Name:               scheduleTask.Name,
						TaskExecutionID:    scheduleTask.TaskExecutionId,
						Input:              scheduleTask.Input,
						ParentTraceContext: currentTraceContext,
					},
				},
				Router: action.Router,
			}
			_ = AddEvent(s, scheduledEvent)
			s.PendingTasks = append(s.PendingTasks, scheduledEvent)
		} else if createSO := action.GetCreateChildWorkflow(); createSO != nil {
			// Autogenerate an instance ID for the sub-orchestration if none is provided, using a
			// deterministic algorithm based on the parent instance ID to help enable de-duplication.
			if createSO.InstanceID == "" {
				createSO.InstanceID = fmt.Sprintf("%s:%04x", s.InstanceID, action.Id)
			}
			_ = AddEvent(s, &protos.HistoryEvent{
				EventID:   ptr.Of(action.Id),
				Timestamp: timestamppb.New(time.Now()),
				EventType: &protos.HistoryEvent_ChildWorkflowInstanceCreated{
					ChildWorkflowInstanceCreated: &protos.ChildWorkflowInstanceCreatedEvent{
						Name:               createSO.Name,
						Input:              createSO.Input,
						InstanceID:         createSO.InstanceID,
						ParentTraceContext: currentTraceContext,
					},
				},
				Router: action.Router,
			})
			startEvent := &protos.HistoryEvent{
				EventID:   ptr.Of(int32(-1)),
				Timestamp: timestamppb.New(time.Now()),
				EventType: &protos.HistoryEvent_ExecutionStarted{
					ExecutionStarted: &protos.ExecutionStartedEvent{
						Name: createSO.Name,
						ParentInstance: &protos.ParentInstanceInfo{
							TaskScheduledId:  action.Id,
							Name:             wrapperspb.String(s.StartEvent.Name),
							WorkflowInstance: &protos.WorkflowInstance{InstanceID: string(s.InstanceID)},
							AppID:            ptr.Of(action.Router.GetSourceAppID()),
						},
						Input: createSO.Input,
						WorkflowInstance: &protos.WorkflowInstance{
							InstanceID:  createSO.InstanceID,
							ExecutionId: wrapperspb.String(uuid.New().String()),
						},
						ParentTraceContext: currentTraceContext,
					},
				},
				Router: action.Router,
			}

			s.PendingMessages = append(s.PendingMessages, &protos.WorkflowRuntimeStateMessage{HistoryEvent: startEvent, TargetInstanceID: createSO.InstanceID})
		} else if terminate := action.GetTerminateWorkflow(); terminate != nil {
			// Send a message to terminate the target orchestration
			msg := &protos.WorkflowRuntimeStateMessage{
				TargetInstanceID: terminate.InstanceID,
				HistoryEvent: &protos.HistoryEvent{
					EventID:   ptr.Of(int32(-1)),
					Timestamp: timestamppb.Now(),
					EventType: &protos.HistoryEvent_ExecutionTerminated{
						ExecutionTerminated: &protos.ExecutionTerminatedEvent{
							Input:   terminate.Reason,
							Recurse: terminate.Recurse,
						},
					},
					Router: action.Router,
				},
			}
			s.PendingMessages = append(s.PendingMessages, msg)
		} else if versionNotAvailable := action.GetWorkflowVersionNotAvailable(); versionNotAvailable != nil {
			versionName := ""
			for _, e := range s.OldEvents {
				if es := e.GetWorkflowStarted(); es != nil {
					versionName = es.GetVersion().GetName()
					break
				}
			}

			msg := &protos.WorkflowRuntimeStateMessage{
				HistoryEvent: &protos.HistoryEvent{
					EventID:   ptr.Of(int32(-1)),
					Timestamp: timestamppb.Now(),
					EventType: &protos.HistoryEvent_ExecutionStalled{
						ExecutionStalled: &protos.ExecutionStalledEvent{
							Reason:      protos.StalledReason_VERSION_NOT_AVAILABLE,
							Description: ptr.Of(fmt.Sprintf("Version not available: %s", versionName)),
						},
					},
					Router: action.Router,
				},
			}
			s.PendingMessages = append(s.PendingMessages, msg)
		} else {
			return false, fmt.Errorf("unknown action type: %v", action)
		}
	}

	return false, nil
}
