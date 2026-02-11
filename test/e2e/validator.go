package e2e

import (
	"context"
)

// ClusterValidator composes multiple checkers for comprehensive cluster state
// validation.
type ClusterValidator struct {
	checkers []Checker
}

// RunAll executes all checkers sequentially and returns all results. It does
// not short-circuit on failure, collecting all validation errors for
// comprehensive reporting.
func (v *ClusterValidator) RunAll(ctx context.Context) []Result {
	results := make([]Result, 0, len(v.checkers))
	for _, checker := range v.checkers {
		results = append(results, checker.Check(ctx))
	}
	return results
}

// NewClusterValidator creates a validator with the specified checkers.
func NewClusterValidator(checkers ...Checker) *ClusterValidator {
	return &ClusterValidator{checkers: checkers}
}
