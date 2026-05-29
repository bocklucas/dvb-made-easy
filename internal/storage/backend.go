package storage

import (
	"context"
	"io"
	"regexp"
	"time"
)

type BackupFile struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	IsEncrypted  bool      `json:"is_encrypted"`
}

type Backend interface {
	ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error)
	Download(ctx context.Context, key string, w io.Writer) error
	TestConnection(ctx context.Context) error
}

type BackendType string

const (
	BackendLocal   BackendType = "local"
	BackendSMB     BackendType = "smb"
	BackendS3      BackendType = "s3"
	BackendWebDAV  BackendType = "webdav"
	BackendAzure   BackendType = "azure"
	BackendDropbox BackendType = "dropbox"
	BackendGDrive  BackendType = "gdrive"
	BackendSFTP    BackendType = "sftp"
)

type Credentials struct {
	Type           BackendType   `json:"type"`
	SavedBackendID string        `json:"saved_backend_id,omitempty"`
	Local          *LocalCreds   `json:"local,omitempty"`
	SMB            *SMBCreds     `json:"smb,omitempty"`
	S3             *S3Creds      `json:"s3,omitempty"`
	WebDAV         *WebDAVCreds  `json:"webdav,omitempty"`
	Azure          *AzureCreds   `json:"azure,omitempty"`
	Dropbox        *DropboxCreds `json:"dropbox,omitempty"`
	GDrive         *GDriveCreds  `json:"gdrive,omitempty"`
	SFTP           *SFTPCreds    `json:"sftp,omitempty"`
}

func (c *Credentials) Sanitize() Credentials {
	if c == nil {
		return Credentials{}
	}
	res := *c
	if res.SMB != nil {
		smbCopy := *res.SMB
		smbCopy.Password = ""
		res.SMB = &smbCopy
	}
	if res.S3 != nil {
		s3Copy := *res.S3
		s3Copy.SecretKey = ""
		res.S3 = &s3Copy
	}
	if res.WebDAV != nil {
		webdavCopy := *res.WebDAV
		webdavCopy.Password = ""
		res.WebDAV = &webdavCopy
	}
	if res.Azure != nil {
		azureCopy := *res.Azure
		azureCopy.ConnectionString = ""
		res.Azure = &azureCopy
	}
	if res.Dropbox != nil {
		dropboxCopy := *res.Dropbox
		dropboxCopy.AccessToken = ""
		dropboxCopy.AppSecret = ""
		res.Dropbox = &dropboxCopy
	}
	if res.GDrive != nil {
		gdriveCopy := *res.GDrive
		gdriveCopy.Credentials = ""
		res.GDrive = &gdriveCopy
	}
	if res.SFTP != nil {
		sftpCopy := *res.SFTP
		sftpCopy.Password = ""
		sftpCopy.PrivateKey = ""
		res.SFTP = &sftpCopy
	}
	return res
}


func (c *Credentials) HasSecrets() bool {
	if c == nil {
		return false
	}
	if c.SMB != nil && c.SMB.Password != "" {
		return true
	}
	if c.S3 != nil && c.S3.SecretKey != "" {
		return true
	}
	if c.WebDAV != nil && c.WebDAV.Password != "" {
		return true
	}
	if c.Azure != nil && c.Azure.ConnectionString != "" {
		return true
	}
	if c.Dropbox != nil && (c.Dropbox.AccessToken != "" || c.Dropbox.AppSecret != "") {
		return true
	}
	if c.GDrive != nil && c.GDrive.Credentials != "" {
		return true
	}
	if c.SFTP != nil && (c.SFTP.Password != "" || c.SFTP.PrivateKey != "") {
		return true
	}
	return false
}

type LocalCreds struct {
	Path string `json:"path"`
}

type SMBCreds struct {
	Host     string `json:"host"`
	Share    string `json:"share"`
	Path     string `json:"path"`
	Username string `json:"username"`
	Password string `json:"password"`
	Port     int    `json:"port"`
}

type S3Creds struct {
	Bucket       string `json:"bucket"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	StorageClass string `json:"storage_class"`
}

type WebDAVCreds struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Path     string `json:"path"`
	Insecure bool   `json:"insecure"`
}

type AzureCreds struct {
	ConnectionString string `json:"connection_string"`
	Container        string `json:"container"`
}

type DropboxCreds struct {
	AccessToken string `json:"access_token"`
	AppKey      string `json:"app_key"`
	AppSecret   string `json:"app_secret"`
	RemotePath  string `json:"remote_path"`
}

type GDriveCreds struct {
	FolderID    string `json:"folder_id"`
	Credentials string `json:"credentials"` // Service account JSON string
	Impersonate string `json:"impersonate"`
}

type SFTPCreds struct {
	Host       string `json:"host"`
	User       string `json:"user"`
	Port       int    `json:"port"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	RemotePath string `json:"remote_path"`
	HostKey    string `json:"host_key,omitempty"`
}

