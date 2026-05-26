package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type DropboxBackend struct {
	accessToken string
	remotePath  string
	apiURL      string
	contentURL  string
	client      *http.Client
}

func NewDropboxBackend(creds *DropboxCreds) (*DropboxBackend, error) {
	if creds == nil || creds.AccessToken == "" {
		return nil, fmt.Errorf("dropbox credentials: access_token is required")
	}

	return &DropboxBackend{
		accessToken: creds.AccessToken,
		remotePath:  cleanDropboxPath(creds.RemotePath),
		apiURL:      "https://api.dropboxapi.com",
		contentURL:  "https://content.dropboxapi.com",
		client:      &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func cleanDropboxPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

type dropboxEntry struct {
	Tag            string    `json:".tag"`
	Name           string    `json:"name"`
	PathDisplay    string    `json:"path_display"`
	Size           int64     `json:"size"`
	ClientModified time.Time `json:"client_modified"`
}

type dropboxListFolderResponse struct {
	Entries []dropboxEntry `json:"entries"`
	Cursor  string         `json:"cursor"`
	HasMore bool           `json:"has_more"`
}

func (b *DropboxBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	var backups []BackupFile

	// Initial request
	listReq := map[string]interface{}{
		"path":      b.remotePath,
		"recursive": false,
	}
	reqBody, err := json.Marshal(listReq)
	if err != nil {
		return nil, fmt.Errorf("dropbox marshal request: %w", err)
	}

	url := b.apiURL + "/2/files/list_folder"
	respData, err := b.postRequest(ctx, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dropbox list_folder: %w", err)
	}

	var listResp dropboxListFolderResponse
	if err := json.Unmarshal(respData, &listResp); err != nil {
		return nil, fmt.Errorf("dropbox unmarshal response: %w", err)
	}

	for _, entry := range listResp.Entries {
		b.processEntry(&backups, entry, pattern)
	}

	// Paginate if has_more is true
	for listResp.HasMore {
		contReq := map[string]string{
			"cursor": listResp.Cursor,
		}
		reqBody, err = json.Marshal(contReq)
		if err != nil {
			return nil, fmt.Errorf("dropbox marshal continue request: %w", err)
		}

		url = b.apiURL + "/2/files/list_folder/continue"
		respData, err = b.postRequest(ctx, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("dropbox list_folder/continue: %w", err)
		}

		listResp = dropboxListFolderResponse{}
		if err := json.Unmarshal(respData, &listResp); err != nil {
			return nil, fmt.Errorf("dropbox unmarshal continue response: %w", err)
		}

		for _, entry := range listResp.Entries {
			b.processEntry(&backups, entry, pattern)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *DropboxBackend) processEntry(backups *[]BackupFile, entry dropboxEntry, pattern *regexp.Regexp) {
	if entry.Tag != "file" {
		return
	}

	match, encrypted := MatchBackupPattern(entry.Name, pattern)
	if !match {
		return
	}

	*backups = append(*backups, BackupFile{
		Key:          entry.PathDisplay,
		Size:         entry.Size,
		LastModified: entry.ClientModified,
		IsEncrypted:  encrypted,
	})
}

func (b *DropboxBackend) Download(ctx context.Context, key string, w io.Writer) error {
	url := b.contentURL + "/2/files/download"
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("dropbox create download request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+b.accessToken)

	arg := map[string]string{"path": key}
	argJSON, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("dropbox marshal download args: %w", err)
	}
	req.Header.Set("Dropbox-API-Arg", string(argJSON))

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("dropbox execute download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dropbox download failed with status %d: %s", resp.StatusCode, string(body))
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("copy dropbox download stream: %w", err)
	}

	return nil
}

func (b *DropboxBackend) TestConnection(ctx context.Context) error {
	url := b.apiURL + "/2/users/get_current_account"
	_, err := b.postRequest(ctx, url, []byte("null"))
	if err != nil {
		return fmt.Errorf("dropbox test connection: %w", err)
	}
	return nil
}

func (b *DropboxBackend) postRequest(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+b.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
