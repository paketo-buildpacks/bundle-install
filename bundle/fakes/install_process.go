package fakes

import "sync"

type InstallProcess struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			LayerPath  string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *InstallProcess) Execute(param1 string, param2 string) error {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.WorkingDir = param1
	f.ExecuteCall.Receives.LayerPath = param2
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2)
	}
	return f.ExecuteCall.Returns.Error
}
