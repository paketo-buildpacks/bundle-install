package fakes

import "sync"

type VersionResolver struct {
	CompareMajorMinorCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			String_1 string
			String_2 string
		}
		Returns struct {
			Bool  bool
			Error error
		}
		Stub func(string, string) (bool, error)
	}
	LookupCall struct {
		sync.Mutex
		CallCount int
		Returns   struct {
			Version string
			Err     error
		}
		Stub func() (string, error)
	}
}

func (f *VersionResolver) CompareMajorMinor(param1 string, param2 string) (bool, error) {
	f.CompareMajorMinorCall.Lock()
	defer f.CompareMajorMinorCall.Unlock()
	f.CompareMajorMinorCall.CallCount++
	f.CompareMajorMinorCall.Receives.String_1 = param1
	f.CompareMajorMinorCall.Receives.String_2 = param2
	if f.CompareMajorMinorCall.Stub != nil {
		return f.CompareMajorMinorCall.Stub(param1, param2)
	}
	return f.CompareMajorMinorCall.Returns.Bool, f.CompareMajorMinorCall.Returns.Error
}
func (f *VersionResolver) Lookup() (string, error) {
	f.LookupCall.Lock()
	defer f.LookupCall.Unlock()
	f.LookupCall.CallCount++
	if f.LookupCall.Stub != nil {
		return f.LookupCall.Stub()
	}
	return f.LookupCall.Returns.Version, f.LookupCall.Returns.Err
}
