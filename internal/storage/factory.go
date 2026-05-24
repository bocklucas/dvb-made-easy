package storage

import "fmt"

func NewBackend(creds *Credentials) (Backend, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials must not be nil")
	}

	switch creds.Type {
	case BackendLocal:
		return NewLocalBackend(creds.Local)
	case BackendSMB:
		return NewSMBBackend(creds.SMB)
	default:
		return nil, fmt.Errorf("unsupported backend type: %q", creds.Type)
	}
}
