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
	case BackendS3:
		return NewS3Backend(creds.S3)
	case BackendWebDAV:
		return NewWebDAVBackend(creds.WebDAV)
	case BackendAzure:
		return NewAzureBackend(creds.Azure)
	case BackendDropbox:
		return NewDropboxBackend(creds.Dropbox)
	case BackendGDrive:
		return NewGDriveBackend(creds.GDrive)
	case BackendSFTP:
		return NewSFTPBackend(creds.SFTP)
	default:
		return nil, fmt.Errorf("unsupported backend type: %q", creds.Type)
	}
}

