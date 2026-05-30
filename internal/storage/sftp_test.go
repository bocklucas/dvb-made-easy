package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func generatePrivateKey() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, privateKeyPEM); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestSFTPBackend(t *testing.T) {
	privateBytes, err := generatePrivateKey()
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "testuser" && string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("auth failed")
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portStr, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portStr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()

				sConn, chans, reqs, err := ssh.NewServerConn(conn, config)
				if err != nil {
					return
				}
				defer sConn.Close()

				go ssh.DiscardRequests(reqs)

				for newChan := range chans {
					if newChan.ChannelType() != "session" {
						newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
						continue
					}

					channel, requests, err := newChan.Accept()
					if err != nil {
						continue
					}

					go func(in <-chan *ssh.Request) {
						for req := range in {
							ok := false
							if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
								ok = true
							}
							req.Reply(ok, nil)
							if ok {
								server, err := sftp.NewServer(channel)
								if err == nil {
									server.Serve()
								}
								channel.Close()
							}
						}
					}(requests)
				}
			}(conn)
		}
	}()

	// Setup mock files in temp dir
	tmpDir := t.TempDir()
	backupFilePath := filepath.Join(tmpDir, "backup-db-2026-05-25T12-00-00.tar.gz")
	err = os.WriteFile(backupFilePath, []byte("mock sftp payload"), 0600)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}
	otherFilePath := filepath.Join(tmpDir, "other.txt")
	err = os.WriteFile(otherFilePath, []byte("other payload"), 0600)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}

	backend, err := NewSFTPBackend(&SFTPCreds{
		Host:       "127.0.0.1",
		Port:       port,
		User:       "testuser",
		Password:   "testpass",
		RemotePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Test Connection
	err = backend.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}

	// Test ListBackups
	pattern := regexp.MustCompile(`^backup-db-.*\.tar\.gz$`)
	backups, err := backend.ListBackups(context.Background(), pattern)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}

	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	expectedKey := filepath.Join(tmpDir, "backup-db-2026-05-25T12-00-00.tar.gz")
	if backups[0].Key != expectedKey {
		t.Errorf("expected key %s, got %s", expectedKey, backups[0].Key)
	}

	// Test Download
	var buf bytes.Buffer
	err = backend.Download(context.Background(), expectedKey, &buf)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if buf.String() != "mock sftp payload" {
		t.Errorf("expected 'mock sftp payload', got %q", buf.String())
	}
}
