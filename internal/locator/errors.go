package locator

import (
	"errors"
	"fmt"
)

// ErrorType categorizes locator errors for intelligent error handling
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota

	// Transient errors - retry with backoff
	ErrorTypeNetworkTransient // Network timeout, connection reset
	ErrorTypeAPITransient     // API timeout, server error (5xx)

	// Permanent errors - fail fast or give up after few retries
	ErrorTypeResourceNotFound  // Pod, Service, Deployment doesn't exist
	ErrorTypePodNotRunning     // Pod exists but not in Running state
	ErrorTypePodFailed         // Pod in Failed state
	ErrorTypeConfigInvalid     // Invalid configuration (port, selector, etc)
	ErrorTypePermissionDenied  // No permission to access resource
	ErrorTypeNoPodAvailable    // No running pods available for resource (might retry longer)
)

// LocateError wraps location errors with type information for intelligent retry handling
type LocateError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error implements the error interface
func (e *LocateError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements error unwrapping for error chains
func (e *LocateError) Unwrap() error {
	return e.Err
}

// IsLocateError checks if an error is a LocateError
func IsLocateError(err error) bool {
	var locErr *LocateError
	return errors.As(err, &locErr)
}

// GetErrorType extracts the error type from a LocateError
func GetErrorType(err error) ErrorType {
	var locErr *LocateError
	if errors.As(err, &locErr) {
		return locErr.Type
	}
	return ErrorTypeUnknown
}

// NewResourceNotFoundError creates an error for missing resources
func NewResourceNotFoundError(resourceType, name string, err error) error {
	return &LocateError{
		Type:    ErrorTypeResourceNotFound,
		Message: fmt.Sprintf("%s %s not found", resourceType, name),
		Err:     err,
	}
}

// NewPodNotRunningError creates an error for non-running pods
func NewPodNotRunningError(podName, phase string, err error) error {
	return &LocateError{
		Type:    ErrorTypePodNotRunning,
		Message: fmt.Sprintf("pod %s is not running (phase: %s)", podName, phase),
		Err:     err,
	}
}

// NewPodFailedError creates an error for failed pods
func NewPodFailedError(podName string, err error) error {
	return &LocateError{
		Type:    ErrorTypePodFailed,
		Message: fmt.Sprintf("pod %s is in failed state", podName),
		Err:     err,
	}
}

// NewConfigInvalidError creates an error for invalid configuration
func NewConfigInvalidError(msg string, err error) error {
	return &LocateError{
		Type:    ErrorTypeConfigInvalid,
		Message: msg,
		Err:     err,
	}
}

// NewPermissionDeniedError creates an error for permission issues
func NewPermissionDeniedError(operation, resource string, err error) error {
	return &LocateError{
		Type:    ErrorTypePermissionDenied,
		Message: fmt.Sprintf("permission denied: cannot %s %s", operation, resource),
		Err:     err,
	}
}

// NewAPITransientError creates an error for transient API issues
func NewAPITransientError(msg string, err error) error {
	return &LocateError{
		Type:    ErrorTypeAPITransient,
		Message: msg,
		Err:     err,
	}
}

// NewNetworkTransientError creates an error for transient network issues
func NewNetworkTransientError(msg string, err error) error {
	return &LocateError{
		Type:    ErrorTypeNetworkTransient,
		Message: msg,
		Err:     err,
	}
}
