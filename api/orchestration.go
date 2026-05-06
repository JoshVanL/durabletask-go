package api

import (
	"encoding/json"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/kit/ptr"
)

type OrchestrationStatus = protos.OrchestrationStatus

const (
	RUNTIME_STATUS_RUNNING          OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_RUNNING
	RUNTIME_STATUS_COMPLETED        OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_COMPLETED
	RUNTIME_STATUS_CONTINUED_AS_NEW OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_CONTINUED_AS_NEW
	RUNTIME_STATUS_FAILED           OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_FAILED
	RUNTIME_STATUS_CANCELED         OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_CANCELED
	RUNTIME_STATUS_TERMINATED       OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_TERMINATED
	RUNTIME_STATUS_PENDING          OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_PENDING
	RUNTIME_STATUS_SUSPENDED        OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_SUSPENDED
	RUNTIME_STATUS_STALLED          OrchestrationStatus = protos.OrchestrationStatus_ORCHESTRATION_STATUS_STALLED
)

// InstanceID is a unique identifier for a workflow instance.
type InstanceID string

func (i InstanceID) String() string {
	return string(i)
}

// NewWorkflowOptions configures options for starting a new workflow.
type NewWorkflowOptions func(*protos.CreateInstanceRequest) error

// GetWorkflowMetadataOptions is a set of options for fetching workflow metadata.
type FetchWorkflowMetadataOptions func(*protos.GetInstanceRequest)

// RaiseEventOptions is a set of options for raising a workflow event.
type RaiseEventOptions func(*protos.RaiseEventRequest) error

// TerminateOptions is a set of options for terminating a workflow.
type TerminateOptions func(*protos.TerminateRequest) error

// PurgeOptions is a set of options for purging a workflow.
type PurgeOptions func(*protos.PurgeInstancesRequest) error

type RerunOptions func(*protos.RerunWorkflowFromEventRequest) error

// SuspendOptions configures a suspend workflow request.
type SuspendOptions func(*protos.SuspendRequest) error

// ResumeOptions configures a resume workflow request.
type ResumeOptions func(*protos.ResumeRequest) error

type ListInstanceIDsOptions func(*protos.ListInstanceIDsRequest) error

type GetInstanceHistoryOptions func(*protos.GetInstanceHistoryRequest) error

// WithInstanceID configures an explicit workflow instance ID. If not specified,
// a random UUID value will be used for the workflow instance ID.
func WithInstanceID(id InstanceID) NewWorkflowOptions {
	return func(req *protos.CreateInstanceRequest) error {
		req.InstanceId = string(id)
		return nil
	}
}

// WithInput configures an input for the workflow. The specified input must be serializable.
// Proto message types are serialized with protojson; all other types use encoding/json.
func WithInput(input any) NewWorkflowOptions {
	return func(req *protos.CreateInstanceRequest) error {
		bytes, err := marshalData(input)
		if err != nil {
			return err
		}
		req.Input = wrapperspb.String(string(bytes))
		return nil
	}
}

// WithRawInput configures an input for the workflow. The specified input must be a string.
func WithRawInput(rawInput *wrapperspb.StringValue) NewWorkflowOptions {
	return func(req *protos.CreateInstanceRequest) error {
		req.Input = rawInput
		return nil
	}
}

// WithStartTime configures a start time at which the workflow should start running.
// Note that the actual start time could be later than the specified start time if the
// task hub is under load or if the app is not running at the specified start time.
func WithStartTime(startTime time.Time) NewWorkflowOptions {
	return func(req *protos.CreateInstanceRequest) error {
		req.ScheduledStartTimestamp = timestamppb.New(startTime)
		return nil
	}
}

// WithFetchPayloads configures whether to load workflow inputs, outputs, and custom status values, which could be large.
func WithFetchPayloads(fetchPayloads bool) FetchWorkflowMetadataOptions {
	return func(req *protos.GetInstanceRequest) {
		req.GetInputsAndOutputs = fetchPayloads
	}
}

// WithEventPayload configures an event payload. The specified payload must be serializable.
func WithEventPayload(data any) RaiseEventOptions {
	return func(req *protos.RaiseEventRequest) error {
		bytes, err := marshalData(data)
		if err != nil {
			return err
		}
		req.Input = wrapperspb.String(string(bytes))
		return nil
	}
}

// WithRawEventData configures an event payload that is a raw, unprocessed string (e.g. JSON data).
func WithRawEventData(data *wrapperspb.StringValue) RaiseEventOptions {
	return func(req *protos.RaiseEventRequest) error {
		req.Input = data
		return nil
	}
}

// WithOutput configures an output for the terminated workflow. The specified output must be serializable.
func WithOutput(data any) TerminateOptions {
	return func(req *protos.TerminateRequest) error {
		bytes, err := marshalData(data)
		if err != nil {
			return err
		}
		req.Output = wrapperspb.String(string(bytes))
		return nil
	}
}

// WithRawOutput configures a raw, unprocessed output (i.e. pre-serialized) for the terminated workflow.
func WithRawOutput(data *wrapperspb.StringValue) TerminateOptions {
	return func(req *protos.TerminateRequest) error {
		req.Output = data
		return nil
	}
}

// WithRecursiveTerminate configures whether to terminate all child workflows created by the target workflow.
func WithRecursiveTerminate(recursive bool) TerminateOptions {
	return func(req *protos.TerminateRequest) error {
		req.Recursive = recursive
		return nil
	}
}

// WithRecursivePurge configures whether to purge all child workflows created by the target workflow.
func WithRecursivePurge(recursive bool) PurgeOptions {
	return func(req *protos.PurgeInstancesRequest) error {
		req.Recursive = recursive
		return nil
	}
}

// WithForcePurge configures whether to purge a workflow, regardless of its
// state or if it is processable/being processed. Highly discouraged to use
// unless you know what you are doing.
func WithForcePurge(force bool) PurgeOptions {
	return func(req *protos.PurgeInstancesRequest) error {
		req.Force = &force
		return nil
	}
}

func WorkflowMetadataIsRunning(o *protos.WorkflowMetadata) bool {
	return !WorkflowMetadataIsComplete(o)
}

func WorkflowMetadataIsComplete(o *protos.WorkflowMetadata) bool {
	return o.GetRuntimeStatus() == protos.OrchestrationStatus_ORCHESTRATION_STATUS_COMPLETED ||
		o.GetRuntimeStatus() == protos.OrchestrationStatus_ORCHESTRATION_STATUS_FAILED ||
		o.GetRuntimeStatus() == protos.OrchestrationStatus_ORCHESTRATION_STATUS_TERMINATED ||
		o.GetRuntimeStatus() == protos.OrchestrationStatus_ORCHESTRATION_STATUS_CANCELED
}

func WithRerunInput(input any) RerunOptions {
	return func(req *protos.RerunWorkflowFromEventRequest) error {
		req.OverwriteInput = true

		if input == nil {
			return nil
		}

		bytes, err := marshalData(input)
		if err != nil {
			return err
		}

		req.Input = wrapperspb.String(string(bytes))

		return nil
	}
}

// protoMarshaler uses UseProtoNames so JSON output uses snake_case field names.
var protoMarshaler = protojson.MarshalOptions{UseProtoNames: true}

// marshalData serializes v to JSON. Proto message types use protojson for
// correct handling of well-known types (e.g. google.protobuf.Struct);
// all other types use encoding/json.
func marshalData(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	if msg, ok := v.(proto.Message); ok {
		return protoMarshaler.Marshal(msg)
	}
	return json.Marshal(v)
}

func WithRerunNewInstanceID(id InstanceID) RerunOptions {
	return func(req *protos.RerunWorkflowFromEventRequest) error {
		req.NewInstanceID = ptr.Of(id.String())
		return nil
	}
}

func WithListInstanceIDsPageSize(pageSize uint32) ListInstanceIDsOptions {
	return func(req *protos.ListInstanceIDsRequest) error {
		req.PageSize = &pageSize
		return nil
	}
}

func WithListInstanceIDsContinuationToken(token string) ListInstanceIDsOptions {
	return func(req *protos.ListInstanceIDsRequest) error {
		req.ContinuationToken = &token
		return nil
	}
}

// withRouter merges the supplied appID / namespace into req's router
// fragment, creating one if absent. Used by the WithAppID / WithAppNamespace
// option helpers below to share a single mutation path.
func withRouter[T any](getRouter func(T) *protos.TaskRouter, setRouter func(T, *protos.TaskRouter), req T, appID, namespace *string) {
	r := getRouter(req)
	if r == nil {
		r = &protos.TaskRouter{}
	}
	if appID != nil {
		r.TargetAppID = ptr.Of(*appID)
	}
	if namespace != nil {
		r.TargetAppNamespace = ptr.Of(*namespace)
	}
	setRouter(req, r)
}

// Cross-app / cross-namespace routing helpers. Each operation has its own
// pair of WithXxxAppID / WithXxxAppNamespace because Go's option typing is
// per-operation. Pair WithXxxAppNamespace with the matching WithXxxAppID;
// the namespace alone returns ErrAppNamespaceRequiresAppID.

// Schedule (start) routing.
func WithStartAppID(appID string) NewWorkflowOptions {
	return func(req *protos.CreateInstanceRequest) error {
		withRouter(
			func(r *protos.CreateInstanceRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.CreateInstanceRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithStartAppNamespace(namespace string) NewWorkflowOptions {
	return func(req *protos.CreateInstanceRequest) error {
		withRouter(
			func(r *protos.CreateInstanceRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.CreateInstanceRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}

// Get / fetch metadata routing.
func WithGetAppID(appID string) FetchWorkflowMetadataOptions {
	return func(req *protos.GetInstanceRequest) {
		withRouter(
			func(r *protos.GetInstanceRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.GetInstanceRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
	}
}

func WithGetAppNamespace(namespace string) FetchWorkflowMetadataOptions {
	return func(req *protos.GetInstanceRequest) {
		withRouter(
			func(r *protos.GetInstanceRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.GetInstanceRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
	}
}

// RaiseEvent routing.
func WithRaiseEventAppID(appID string) RaiseEventOptions {
	return func(req *protos.RaiseEventRequest) error {
		withRouter(
			func(r *protos.RaiseEventRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.RaiseEventRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithRaiseEventAppNamespace(namespace string) RaiseEventOptions {
	return func(req *protos.RaiseEventRequest) error {
		withRouter(
			func(r *protos.RaiseEventRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.RaiseEventRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}

// Terminate routing.
func WithTerminateAppID(appID string) TerminateOptions {
	return func(req *protos.TerminateRequest) error {
		withRouter(
			func(r *protos.TerminateRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.TerminateRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithTerminateAppNamespace(namespace string) TerminateOptions {
	return func(req *protos.TerminateRequest) error {
		withRouter(
			func(r *protos.TerminateRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.TerminateRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}

// Purge routing.
func WithPurgeAppID(appID string) PurgeOptions {
	return func(req *protos.PurgeInstancesRequest) error {
		withRouter(
			func(r *protos.PurgeInstancesRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.PurgeInstancesRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithPurgeAppNamespace(namespace string) PurgeOptions {
	return func(req *protos.PurgeInstancesRequest) error {
		withRouter(
			func(r *protos.PurgeInstancesRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.PurgeInstancesRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}

// Rerun routing.
func WithRerunAppID(appID string) RerunOptions {
	return func(req *protos.RerunWorkflowFromEventRequest) error {
		withRouter(
			func(r *protos.RerunWorkflowFromEventRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.RerunWorkflowFromEventRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithRerunAppNamespace(namespace string) RerunOptions {
	return func(req *protos.RerunWorkflowFromEventRequest) error {
		withRouter(
			func(r *protos.RerunWorkflowFromEventRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.RerunWorkflowFromEventRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}

func routerAppID(r *protos.TaskRouter) *string {
	if r == nil {
		return nil
	}
	return r.TargetAppID
}

// Suspend routing.
func WithSuspendAppID(appID string) SuspendOptions {
	return func(req *protos.SuspendRequest) error {
		withRouter(
			func(r *protos.SuspendRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.SuspendRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithSuspendAppNamespace(namespace string) SuspendOptions {
	return func(req *protos.SuspendRequest) error {
		withRouter(
			func(r *protos.SuspendRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.SuspendRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}

// Resume routing.
func WithResumeAppID(appID string) ResumeOptions {
	return func(req *protos.ResumeRequest) error {
		withRouter(
			func(r *protos.ResumeRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.ResumeRequest, v *protos.TaskRouter) { r.Router = v },
			req, &appID, nil)
		return nil
	}
}

func WithResumeAppNamespace(namespace string) ResumeOptions {
	return func(req *protos.ResumeRequest) error {
		withRouter(
			func(r *protos.ResumeRequest) *protos.TaskRouter { return r.GetRouter() },
			func(r *protos.ResumeRequest, v *protos.TaskRouter) { r.Router = v },
			req, nil, &namespace)
		return ValidateAppNamespaceRequiresAppID(routerAppID(req.GetRouter()), &namespace)
	}
}
