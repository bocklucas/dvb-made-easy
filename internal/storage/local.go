package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

type LocalBackend struct {
	path string
}

func NewLocalBackend(creds *LocalCreds) (*LocalBackend, error) {
	if creds == nil || creds.Path == "" {
		return nil, fmt.Errorf("local credentials: path is required")
	}
	return &LocalBackend{path: creds.Path}, nil
}

func (b *LocalBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	entries, err := os.ReadDir(b.path)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", b.path, err)
	}

	var backups []BackupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match, encrypted := MatchBackupPattern(entry.Name(), pattern)
		if !match {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupFile{
			Key:          entry.Name(),
			Size:         info.Size(),
			LastModified: info.ModTime(),
			IsEncrypted:  encrypted,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *LocalBackend) Download(ctx context.Context, key string, w io.Writer) error {
	path := filepath.Join(b.path, filepath.Base(key))
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", key, err)
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy %s: %w", key, err)
	}
	return nil
}

func (b *LocalBackend) TestConnection(ctx context.Context) error {
	info, err := os.Stat(b.path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", b.path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", b.path)
	}
	return nil
}
