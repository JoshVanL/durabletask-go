package api

import (
	"errors"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/kit/ptr"
)

// ErrAppNamespaceRequiresAppID is returned when an option specifies a target
// app namespace without a corresponding target app ID.
var ErrAppNamespaceRequiresAppID = errors.New("workflow option WithAppNamespace requires WithAppID to also be set")

// TaskRouterFromTarget builds the routing envelope used by cross-app and
// cross-namespace workflow operations. Returns nil when the call is local
// (no target app ID). Cross-namespace routing requires both target app ID
// and namespace; callers must validate that invariant via
// ValidateAppNamespaceRequiresAppID before reaching here.
func TaskRouterFromTarget(targetAppID, targetAppNamespace *string) *protos.TaskRouter {
	if targetAppID == nil {
		return nil
	}
	r := &protos.TaskRouter{TargetAppID: ptr.Of(*targetAppID)}
	if targetAppNamespace != nil {
		r.TargetAppNamespace = ptr.Of(*targetAppNamespace)
	}
	return r
}

// ValidateAppNamespaceRequiresAppID enforces that a target namespace must be
// paired with a target app ID. Returns ErrAppNamespaceRequiresAppID when the
// invariant is violated, otherwise nil.
func ValidateAppNamespaceRequiresAppID(targetAppID, targetAppNamespace *string) error {
	if targetAppNamespace != nil && targetAppID == nil {
		return ErrAppNamespaceRequiresAppID
	}
	return nil
}
