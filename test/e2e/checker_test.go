package e2e

import (
	"fmt"
	"testing"

	o "github.com/onsi/gomega"
)

func TestNewResult(t *testing.T) {
	g := o.NewWithT(t)

	r := NewResult("all good")
	g.Expect(r.Passed).To(o.BeTrue())
	g.Expect(r.Message).To(o.Equal("all good"))
}

func TestNewResult_EmptyMessage(t *testing.T) {
	g := o.NewWithT(t)

	r := NewResult("")
	g.Expect(r.Passed).To(o.BeTrue())
	g.Expect(r.Message).To(o.BeEmpty())
}

func TestNewFailedResult(t *testing.T) {
	g := o.NewWithT(t)

	r := NewFailedResult(fmt.Errorf("something broke"))
	g.Expect(r.Passed).To(o.BeFalse())
	g.Expect(r.Message).To(o.Equal("something broke"))
}
