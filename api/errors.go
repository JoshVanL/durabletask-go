package api

import (
	"errors"
	"fmt"
)

var (
	ErrInstanceNotFound  = errors.New("no such instance exists")
	ErrNotStarted        = errors.New("workflow has not started")
	ErrNotCompleted      = errors.New("workflow has not yet completed")
	ErrNoFailures        = errors.New("workflow did not report failure details")
	ErrDuplicateInstance = errors.New("workflow instance already exists")
	ErrIgnoreInstance    = errors.New("ignore creating workflow instance")
	ErrTaskCancelled     = errors.New("task was cancelled")
	ErrStalled           = errors.New("workflow is stalled")
)

type UnknownTaskIDError struct {
	TaskID     int32
	InstanceID string
}

func NewUnknownTaskIDError(instanceID string, taskID int32) error {
	return &UnknownTaskIDError{
		TaskID:     taskID,
		InstanceID: instanceID,
	}
}

func (e *UnknownTaskIDError) Error() string {
	return fmt.Sprintf("unknown instance ID/task ID combo: %s/%d", e.InstanceID, e.TaskID)
}

func IsUnknownTaskIDError(err error) bool {
	_, ok := err.(*UnknownTaskIDError)
	return ok
}

type UnknownInstanceIDError struct {
	InstanceID string
}

func NewUnknownInstanceIDError(instanceID string) error {
	return &UnknownInstanceIDError{
		InstanceID: instanceID,
	}
}

func (e *UnknownInstanceIDError) Error() string {
	return fmt.Sprintf("unknown instance ID: %s", e.InstanceID)
}

func IsUnknownInstanceIDError(err error) bool {
	_, ok := err.(*UnknownInstanceIDError)
	return ok
}

type UnsupportedVersionError struct{}

func NewUnsupportedVersionError() error {
	return &UnsupportedVersionError{}
}

func (e *UnsupportedVersionError) Error() string {
	return "workflow version is not registered"
}

func IsUnsupportedVersionError(err error) bool {
	_, ok := err.(*UnsupportedVersionError)
	return ok
}
