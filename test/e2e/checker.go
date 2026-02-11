package e2e

import "context"

// Checker defines the interface for cluster state validation components.
type Checker interface {
	Check(ctx context.Context) Result
}

// Result represents the outcome of a checker validation.
type Result struct {
	Passed  bool   // true if validation succeeded
	Message string // descriptive message (error details if Passed=false)
}

// NewResult creates a successful result with an optional message.
func NewResult(message string) Result {
	return Result{Passed: true, Message: message}
}

// NewFailedResult creates a failed result with an error message.
func NewFailedResult(err error) Result {
	return Result{Passed: false, Message: err.Error()}
}
