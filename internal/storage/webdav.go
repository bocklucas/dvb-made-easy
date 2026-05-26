package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"

	"github.com/studio-b12/gowebdav"
)

type WebDAVBackend struct {
	client *gowebdav.Client
	path   string
}

func NewWebDAVBackend(creds *WebDAVCreds) (*WebDAVBackend, error) {
	if creds == nil || creds.URL == "" {
		return nil, fmt.Errorf("webdav credentials: url is required")
	}

	client := gowebdav.NewClient(creds.URL, creds.Username, creds.Password)

	if creds.Insecure {
		client.SetTransport(&http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		})
	}

	return &WebDAVBackend{
		client: client,
		path:   creds.Path,
	}, nil
}

func (b *WebDAVBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	if err := b.client.Connect(); err != nil {
		return nil, fmt.Errorf("webdav connect: %w", err)
	}

	dirPath := b.path
	if dirPath == "" {
		dirPath = "/"
	}

	files, err := b.client.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("webdav readdir %s: %w", dirPath, err)
	}

	var backups []BackupFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		match, encrypted := MatchBackupPattern(file.Name(), pattern)
		if !match {
			continue
		}

		key := path.Join(b.path, file.Name())

		backups = append(backups, BackupFile{
			Key:          key,
			Size:         file.Size(),
			LastModified: file.ModTime(),
			IsEncrypted:  encrypted,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *WebDAVBackend) Download(ctx context.Context, key string, w io.Writer) error {
	if err := b.client.Connect(); err != nil {
		return fmt.Errorf("webdav connect: %w", err)
	}

	reader, err := b.client.ReadStream(key)
	if err != nil {
		return fmt.Errorf("webdav readstream %s: %w", key, err)
	}
	defer reader.Close()

	if _, err := io.Copy(w, reader); err != nil {
		return fmt.Errorf("copy webdav body %s: %w", key, err)
	}

	return nil
}

func (b *WebDAVBackend) TestConnection(ctx context.Context) error {
	if err := b.client.Connect(); err != nil {
		return fmt.Errorf("webdav connect: %w", err)
	}

	dirPath := b.path
	if dirPath == "" {
		dirPath = "/"
	}

	_, err := b.client.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("webdav stat %s: %w", dirPath, err)
	}

	return nil
}
