package integration_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/dagger"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	bpDir            string
	bundleInstallURI string
	bundlerURI       string
	mriURI           string
	buildPlanURI     string
)

func TestIntegration(t *testing.T) {
	var (
		Expect = NewWithT(t).Expect
		err    error
	)

	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	bundleInstallURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).NotTo(HaveOccurred())

	// HACK: we need to fix dagger and the package.sh scripts so that this isn't required
	bundleInstallURI = fmt.Sprintf("%s.tgz", bundleInstallURI)

	bundlerURI, err = dagger.GetLatestCommunityBuildpack("cloudfoundry", "bundler-cnb")
	Expect(err).NotTo(HaveOccurred())

	mriURI, err = dagger.GetLatestCommunityBuildpack("cloudfoundry", "mri-cnb")
	Expect(err).NotTo(HaveOccurred())

	buildPlanURI, err = dagger.GetLatestCommunityBuildpack("ForestEckhardt", "build-plan")
	Expect(err).NotTo(HaveOccurred())

	defer func() {
		dagger.DeleteBuildpack(bundleInstallURI)
		dagger.DeleteBuildpack(bundlerURI)
		dagger.DeleteBuildpack(mriURI)
		dagger.DeleteBuildpack(buildPlanURI)
	}()

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("SimpleApp", testSimpleApp)
	suite("Logging", testLogging)

	dagger.SyncParallelOutput(func() { suite.Run(t) })
}

func ContainerLogs(id string) func() string {
	docker := occam.NewDocker()

	return func() string {
		logs, _ := docker.Container.Logs.Execute(id)
		return logs.String()
	}
}

func GetGitVersion() (string, error) {
	gitExec := pexec.NewExecutable("git")
	revListOut := bytes.NewBuffer(nil)

	err := gitExec.Execute(pexec.Execution{
		Args:   []string{"rev-list", "--tags", "--max-count=1"},
		Stdout: revListOut,
	})
	if err != nil {
		return "", err
	}

	stdout := bytes.NewBuffer(nil)
	err = gitExec.Execute(pexec.Execution{
		Args:   []string{"describe", "--tags", strings.TrimSpace(revListOut.String())},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "v")), nil
}
