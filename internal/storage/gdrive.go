package storage

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GDriveBackend struct {
	srv      *drive.Service
	folderID string
}

func NewGDriveBackend(creds *GDriveCreds) (*GDriveBackend, error) {
	if creds == nil || creds.FolderID == "" || creds.Credentials == "" {
		return nil, fmt.Errorf("gdrive credentials: folder_id and credentials (JSON string) are required")
	}

	ctx := context.Background()

	config, err := google.JWTConfigFromJSON([]byte(creds.Credentials), drive.DriveReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("gdrive parse credentials JSON: %w", err)
	}

	if creds.Impersonate != "" {
		config.Subject = creds.Impersonate
	}

	client := config.Client(ctx)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("gdrive create service: %w", err)
	}

	return &GDriveBackend{
		srv:      srv,
		folderID: creds.FolderID,
	}, nil
}

func (b *GDriveBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	query := fmt.Sprintf("'%s' in parents and trashed = false", b.folderID)

	var backups []BackupFile
	err := b.srv.Files.List().
		Q(query).
		Fields("nextPageToken, files(id, name, size, modifiedTime)").
		Pages(ctx, func(fl *drive.FileList) error {
			for _, file := range fl.Files {
				match, encrypted := MatchBackupPattern(file.Name, pattern)
				if !match {
					continue
				}

				var lastModified time.Time
				if file.ModifiedTime != "" {
					t, err := time.Parse(time.RFC3339, file.ModifiedTime)
					if err == nil {
						lastModified = t
					}
				}

				backups = append(backups, BackupFile{
					Key:          file.Name, // Use filename as the key so it renders nicely in the UI
					Size:         file.Size,
					LastModified: lastModified,
					IsEncrypted:  encrypted,
				})
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("gdrive list files: %w", err)
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *GDriveBackend) Download(ctx context.Context, key string, w io.Writer) error {
	query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", key, b.folderID)
	r, err := b.srv.Files.List().Q(query).Fields("files(id)").Do()
	if err != nil {
		return fmt.Errorf("gdrive query file %s: %w", key, err)
	}
	if len(r.Files) == 0 {
		return fmt.Errorf("gdrive file %s not found", key)
	}
	fileID := r.Files[0].Id

	resp, err := b.srv.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("gdrive download file %s (id %s): %w", key, fileID, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("copy gdrive body %s: %w", key, err)
	}
	return nil
}

func (b *GDriveBackend) TestConnection(ctx context.Context) error {
	_, err := b.srv.Files.List().
		Q(fmt.Sprintf("'%s' in parents and trashed = false", b.folderID)).
		PageSize(1).
		Fields("files(id)").
		Do()
	if err != nil {
		return fmt.Errorf("gdrive test connection (list folder %s): %w", b.folderID, err)
	}
	return nil
}
