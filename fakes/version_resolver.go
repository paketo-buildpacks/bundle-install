package fakes

import "sync"

type VersionResolver struct {
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

func (f *VersionResolver) Lookup() (string, error) {
	f.LookupCall.Lock()
	defer f.LookupCall.Unlock()
	f.LookupCall.CallCount++
	if f.LookupCall.Stub != nil {
		return f.LookupCall.Stub()
	}
	return f.LookupCall.Returns.Version, f.LookupCall.Returns.Err
}
