package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
)

type InstallProcess struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			LayerPath  string
			Config     map[string]string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, map[string]string) error
	}
	ShouldRunCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Layer      packit.Layer
			WorkingDir string
		}
		Returns struct {
			Should      bool
			Checksum    string
			RubyVersion string
			Err         error
		}
		Stub func(packit.Layer, string) (bool, string, string, error)
	}
}

func (f *InstallProcess) Execute(param1 string, param2 string, param3 map[string]string) error {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.WorkingDir = param1
	f.ExecuteCall.Receives.LayerPath = param2
	f.ExecuteCall.Receives.Config = param3
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2, param3)
	}
	return f.ExecuteCall.Returns.Error
}
func (f *InstallProcess) ShouldRun(param1 packit.Layer, param2 string) (bool, string, string, error) {
	f.ShouldRunCall.Lock()
	defer f.ShouldRunCall.Unlock()
	f.ShouldRunCall.CallCount++
	f.ShouldRunCall.Receives.Layer = param1
	f.ShouldRunCall.Receives.WorkingDir = param2
	if f.ShouldRunCall.Stub != nil {
		return f.ShouldRunCall.Stub(param1, param2)
	}
	return f.ShouldRunCall.Returns.Should, f.ShouldRunCall.Returns.Checksum, f.ShouldRunCall.Returns.RubyVersion, f.ShouldRunCall.Returns.Err
}
