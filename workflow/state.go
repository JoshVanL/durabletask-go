package workflow

import (
	"github.com/dapr/durabletask-go/api/protos"
)

const (
	StatusRunning        = protos.WorkflowStatus_WORKFLOW_STATUS_RUNNING
	StatusCompleted      = protos.WorkflowStatus_WORKFLOW_STATUS_COMPLETED
	StatusContinuedAsNew = protos.WorkflowStatus_WORKFLOW_STATUS_CONTINUED_AS_NEW
	StatusFailed         = protos.WorkflowStatus_WORKFLOW_STATUS_FAILED
	StatusCanceled       = protos.WorkflowStatus_WORKFLOW_STATUS_CANCELED
	StatusTerminated     = protos.WorkflowStatus_WORKFLOW_STATUS_TERMINATED
	StatusPending        = protos.WorkflowStatus_WORKFLOW_STATUS_PENDING
	StatusSuspended      = protos.WorkflowStatus_WORKFLOW_STATUS_SUSPENDED
	StatusStalled        = protos.WorkflowStatus_WORKFLOW_STATUS_STALLED
)

type WorkflowMetadata protos.WorkflowMetadata
type ListInstanceIDsResponse protos.ListInstanceIDsResponse
type GetInstanceHistoryResponse protos.GetInstanceHistoryResponse

func (w WorkflowMetadata) String() string {
	switch w.RuntimeStatus {
	case protos.WorkflowStatus_WORKFLOW_STATUS_RUNNING:
		return "RUNNING"
	case protos.WorkflowStatus_WORKFLOW_STATUS_COMPLETED:
		return "COMPLETED"
	case protos.WorkflowStatus_WORKFLOW_STATUS_CONTINUED_AS_NEW:
		return "CONTINUED_AS_NEW"
	case protos.WorkflowStatus_WORKFLOW_STATUS_FAILED:
		return "FAILED"
	case protos.WorkflowStatus_WORKFLOW_STATUS_CANCELED:
		return "CANCELED"
	case protos.WorkflowStatus_WORKFLOW_STATUS_TERMINATED:
		return "TERMINATED"
	case protos.WorkflowStatus_WORKFLOW_STATUS_PENDING:
		return "PENDING"
	case protos.WorkflowStatus_WORKFLOW_STATUS_SUSPENDED:
		return "SUSPENDED"
	case protos.WorkflowStatus_WORKFLOW_STATUS_STALLED:
		return "STALLED"
	default:
		return ""
	}
}
