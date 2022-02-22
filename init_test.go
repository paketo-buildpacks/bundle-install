package bundleinstall_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitBundleInstall(t *testing.T) {
	suite := spec.New("bundle-install", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Build", testBuild)
	suite("BundleInstallProcess", testBundleInstallProcess)
	suite("Detect", testDetect)
	suite("GemfileParser", testGemfileParser)
	suite("RubyVersionResolver", testRubyVersionResolver)
	suite.Run(t)
}
