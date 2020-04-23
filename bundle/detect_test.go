package bundle_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("returns a plan that provides node_modules", func() {
		Expect(1).To(Equal(2))
	})
}
