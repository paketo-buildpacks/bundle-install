package fakes

import "sync"

type InstallProcess struct {
	ExecuteCall struct {
		mutex     sync.Mutex
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
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Metadata map[string]interface {
			}
			WorkingDir string
		}
		Returns struct {
			Should      bool
			Checksum    string
			RubyVersion string
			Err         error
		}
		Stub func(map[string]interface {
		}, string) (bool, string, string, error)
	}
}

func (f *InstallProcess) Execute(param1 string, param2 string, param3 map[string]string) error {
	f.ExecuteCall.mutex.Lock()
	defer f.ExecuteCall.mutex.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.WorkingDir = param1
	f.ExecuteCall.Receives.LayerPath = param2
	f.ExecuteCall.Receives.Config = param3
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2, param3)
	}
	return f.ExecuteCall.Returns.Error
}
func (f *InstallProcess) ShouldRun(param1 map[string]interface {
}, param2 string) (bool, string, string, error) {
	f.ShouldRunCall.mutex.Lock()
	defer f.ShouldRunCall.mutex.Unlock()
	f.ShouldRunCall.CallCount++
	f.ShouldRunCall.Receives.Metadata = param1
	f.ShouldRunCall.Receives.WorkingDir = param2
	if f.ShouldRunCall.Stub != nil {
		return f.ShouldRunCall.Stub(param1, param2)
	}
	return f.ShouldRunCall.Returns.Should, f.ShouldRunCall.Returns.Checksum, f.ShouldRunCall.Returns.RubyVersion, f.ShouldRunCall.Returns.Err
}
