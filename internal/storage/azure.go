package storage

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type AzureBackend struct {
	client    *azblob.Client
	container string
}

func NewAzureBackend(creds *AzureCreds) (*AzureBackend, error) {
	if creds == nil || creds.ConnectionString == "" || creds.Container == "" {
		return nil, fmt.Errorf("azure credentials: connection_string and container are required")
	}

	client, err := azblob.NewClientFromConnectionString(creds.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("azure create client: %w", err)
	}

	return &AzureBackend{
		client:    client,
		container: creds.Container,
	}, nil
}

func (b *AzureBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	pager := b.client.NewListBlobsFlatPager(b.container, nil)

	var backups []BackupFile
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure list blobs: %w", err)
		}

		for _, blob := range resp.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			name := *blob.Name
			parts := strings.Split(name, "/")
			filename := parts[len(parts)-1]
			if filename == "" {
				continue
			}

			match, encrypted := MatchBackupPattern(filename, pattern)
			if !match {
				continue
			}

			var size int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				size = *blob.Properties.ContentLength
			}

			var lastModified time.Time
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = *blob.Properties.LastModified
			}

			backups = append(backups, BackupFile{
				Key:          name,
				Size:         size,
				LastModified: lastModified,
				IsEncrypted:  encrypted,
			})
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *AzureBackend) Download(ctx context.Context, key string, w io.Writer) error {
	resp, err := b.client.DownloadStream(ctx, b.container, key, nil)
	if err != nil {
		return fmt.Errorf("azure download stream %s: %w", key, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("copy azure body %s: %w", key, err)
	}

	return nil
}

func (b *AzureBackend) TestConnection(ctx context.Context) error {
	pager := b.client.NewListBlobsFlatPager(b.container, &azblob.ListBlobsFlatOptions{
		MaxResults: func(v int32) *int32 { return &v }(1),
	})
	if pager.More() {
		_, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("azure test connection (list container %s): %w", b.container, err)
		}
	}
	return nil
}
