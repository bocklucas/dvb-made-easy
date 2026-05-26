package storage

import (
	"context"
	"fmt"
	"io"
	"net"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPBackend struct {
	host       string
	port       int
	user       string
	password   string
	privateKey string
	remotePath string
}

func NewSFTPBackend(creds *SFTPCreds) (*SFTPBackend, error) {
	if creds == nil || creds.Host == "" || creds.User == "" {
		return nil, fmt.Errorf("sftp credentials: host and user are required")
	}

	port := creds.Port
	if port == 0 {
		port = 22
	}

	return &SFTPBackend{
		host:       creds.Host,
		port:       port,
		user:       creds.User,
		password:   creds.Password,
		privateKey: creds.PrivateKey,
		remotePath: creds.RemotePath,
	}, nil
}

func (b *SFTPBackend) connect() (*ssh.Client, *sftp.Client, error) {
	var authMethods []ssh.AuthMethod

	if b.privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(b.privateKey))
		if err != nil {
			return nil, nil, fmt.Errorf("parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if b.password != "" {
		authMethods = append(authMethods, ssh.Password(b.password))
	}

	if len(authMethods) == 0 {
		return nil, nil, fmt.Errorf("sftp: either password or private key must be provided")
	}

	config := &ssh.ClientConfig{
		User:            b.user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(b.host, strconv.Itoa(b.port))
	sshConn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, nil, fmt.Errorf("sftp client: %w", err)
	}

	return sshConn, sftpClient, nil
}

func (b *SFTPBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	sshConn, sftpClient, err := b.connect()
	if err != nil {
		return nil, err
	}
	defer sftpClient.Close()
	defer sshConn.Close()

	dirPath := b.remotePath
	if dirPath == "" {
		dirPath = "."
	}

	files, err := sftpClient.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("sftp readdir %s: %w", dirPath, err)
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

		key := path.Join(b.remotePath, file.Name())

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

func (b *SFTPBackend) Download(ctx context.Context, key string, w io.Writer) error {
	sshConn, sftpClient, err := b.connect()
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	defer sshConn.Close()

	f, err := sftpClient.Open(key)
	if err != nil {
		return fmt.Errorf("sftp open %s: %w", key, err)
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy sftp body %s: %w", key, err)
	}

	return nil
}

func (b *SFTPBackend) TestConnection(ctx context.Context) error {
	sshConn, sftpClient, err := b.connect()
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	defer sshConn.Close()

	dirPath := b.remotePath
	if dirPath == "" {
		dirPath = "."
	}

	_, err = sftpClient.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("sftp stat %s: %w", dirPath, err)
	}

	return nil
}
