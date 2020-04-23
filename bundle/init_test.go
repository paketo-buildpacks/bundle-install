package bundle_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitYarn(t *testing.T) {
	suite := spec.New("bundle-install", spec.Report(report.Terminal{}))
	suite("Detect", testDetect)
	suite.Run(t)
}
