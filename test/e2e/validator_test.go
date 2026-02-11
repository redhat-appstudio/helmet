package e2e

import (
	"context"
	"fmt"
	"testing"

	o "github.com/onsi/gomega"
)

// fakeChecker implements Checker for testing.
type fakeChecker struct {
	result Result
}

func (f *fakeChecker) Check(_ context.Context) Result {
	return f.result
}

func TestClusterValidator_RunAll(t *testing.T) {
	g := o.NewWithT(t)
	ctx := context.Background()

	t.Run("all checkers pass", func(t *testing.T) {
		v := NewClusterValidator(
			&fakeChecker{result: NewResult("check-1 ok")},
			&fakeChecker{result: NewResult("check-2 ok")},
		)
		results := v.RunAll(ctx)

		g.Expect(results).To(o.HaveLen(2))
		g.Expect(results[0].Passed).To(o.BeTrue())
		g.Expect(results[1].Passed).To(o.BeTrue())
	})

	t.Run("collects all failures without short-circuiting", func(t *testing.T) {
		v := NewClusterValidator(
			&fakeChecker{result: NewFailedResult(fmt.Errorf("fail-1"))},
			&fakeChecker{result: NewResult("check-2 ok")},
			&fakeChecker{result: NewFailedResult(fmt.Errorf("fail-3"))},
		)
		results := v.RunAll(ctx)

		g.Expect(results).To(o.HaveLen(3))
		g.Expect(results[0].Passed).To(o.BeFalse())
		g.Expect(results[0].Message).To(o.Equal("fail-1"))
		g.Expect(results[1].Passed).To(o.BeTrue())
		g.Expect(results[2].Passed).To(o.BeFalse())
		g.Expect(results[2].Message).To(o.Equal("fail-3"))
	})

	t.Run("empty validator returns no results", func(t *testing.T) {
		v := NewClusterValidator()
		results := v.RunAll(ctx)

		g.Expect(results).To(o.BeEmpty())
	})

	t.Run("single checker", func(t *testing.T) {
		v := NewClusterValidator(
			&fakeChecker{result: NewResult("only check")},
		)
		results := v.RunAll(ctx)

		g.Expect(results).To(o.HaveLen(1))
		g.Expect(results[0].Passed).To(o.BeTrue())
		g.Expect(results[0].Message).To(o.Equal("only check"))
	})
}
