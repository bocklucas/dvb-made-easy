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
	BackendLocal BackendType = "local"
	BackendSMB   BackendType = "smb"
)

type Credentials struct {
	Type  BackendType `json:"type"`
	Local *LocalCreds `json:"local,omitempty"`
	SMB   *SMBCreds   `json:"smb,omitempty"`
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
