package storage

import (
	"context"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"

	"github.com/hirochachacha/go-smb2"
)

const defaultSMBPort = 445

type SMBBackend struct {
	host     string
	port     int
	share    string
	path     string
	username string
	password string
}

func NewSMBBackend(creds *SMBCreds) (*SMBBackend, error) {
	if creds == nil {
		return nil, fmt.Errorf("smb credentials: must not be nil")
	}
	if creds.Host == "" {
		return nil, fmt.Errorf("smb credentials: host is required")
	}
	if creds.Share == "" {
		return nil, fmt.Errorf("smb credentials: share is required")
	}
	port := creds.Port
	if port == 0 {
		port = defaultSMBPort
	}
	return &SMBBackend{
		host:     creds.Host,
		port:     port,
		share:    creds.Share,
		path:     creds.Path,
		username: creds.Username,
		password: creds.Password,
	}, nil
}

func (b *SMBBackend) connect(ctx context.Context) (*smb2.Share, net.Conn, *smb2.Session, error) {
	addr := fmt.Sprintf("%s:%d", b.host, b.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     b.username,
			Password: b.password,
		},
	}

	session, err := d.DialContext(ctx, conn)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("smb session: %w", err)
	}

	share, err := session.Mount(b.share)
	if err != nil {
		session.Logoff()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("mount share %s: %w", b.share, err)
	}

	return share, conn, session, nil
}

func (b *SMBBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	share, conn, session, err := b.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer session.Logoff()

	dir := b.path
	if dir == "" {
		dir = "."
	}

	entries, err := share.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
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
		backups = append(backups, BackupFile{
			Key:          entry.Name(),
			Size:         entry.Size(),
			LastModified: entry.ModTime(),
			IsEncrypted:  encrypted,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *SMBBackend) Download(ctx context.Context, key string, w io.Writer) error {
	share, conn, session, err := b.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer session.Logoff()

	path := key
	if b.path != "" {
		path = b.path + "/" + key
	}

	f, err := share.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", key, err)
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy %s: %w", key, err)
	}
	return nil
}

func (b *SMBBackend) TestConnection(ctx context.Context) error {
	share, conn, session, err := b.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer session.Logoff()

	dir := b.path
	if dir == "" {
		dir = "."
	}

	_, err = share.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat %s: %w", dir, err)
	}
	return nil
}
