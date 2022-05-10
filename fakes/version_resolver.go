package fakes

import "sync"

type VersionResolver struct {
	CompareMajorMinorCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Left  string
			Right string
		}
		Returns struct {
			Bool  bool
			Error error
		}
		Stub func(string, string) (bool, error)
	}
	LookupCall struct {
		mutex     sync.Mutex
		CallCount int
		Returns   struct {
			Version string
			Err     error
		}
		Stub func() (string, error)
	}
}

func (f *VersionResolver) CompareMajorMinor(param1 string, param2 string) (bool, error) {
	f.CompareMajorMinorCall.mutex.Lock()
	defer f.CompareMajorMinorCall.mutex.Unlock()
	f.CompareMajorMinorCall.CallCount++
	f.CompareMajorMinorCall.Receives.Left = param1
	f.CompareMajorMinorCall.Receives.Right = param2
	if f.CompareMajorMinorCall.Stub != nil {
		return f.CompareMajorMinorCall.Stub(param1, param2)
	}
	return f.CompareMajorMinorCall.Returns.Bool, f.CompareMajorMinorCall.Returns.Error
}
func (f *VersionResolver) Lookup() (string, error) {
	f.LookupCall.mutex.Lock()
	defer f.LookupCall.mutex.Unlock()
	f.LookupCall.CallCount++
	if f.LookupCall.Stub != nil {
		return f.LookupCall.Stub()
	}
	return f.LookupCall.Returns.Version, f.LookupCall.Returns.Err
}
