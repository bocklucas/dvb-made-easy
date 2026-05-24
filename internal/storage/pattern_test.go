package storage_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/storage"
)

func TestCompileBackupPattern(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		match   string
		noMatch string
	}{
		{
			"data prefix",
			"backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"backup-data-2024-01-15T10-30-00.tar.gz",
			"backup-config-2024-01-15T10-30-00.tar.gz",
		},
		{
			"config prefix",
			"backup-config-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"backup-config-2024-01-15T10-30-00.tar.gz",
			"backup-data-2024-01-15T10-30-00.tar.gz",
		},
		{
			"encrypted match",
			"backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"backup-data-2024-01-15T10-30-00.tar.gz.gpg",
			"",
		},
		{
			"no space extension",
			"backup-data-%Y-%m-%dT%H-%M-%S.{{.Extension}}",
			"backup-data-2024-01-15T10-30-00.tar.gz",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := storage.CompileBackupPattern(tt.format)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			match, _ := storage.MatchBackupPattern(tt.match, pattern)
			if !match {
				t.Errorf("%q should match pattern from %q", tt.match, tt.format)
			}
			if tt.noMatch != "" {
				noMatch, _ := storage.MatchBackupPattern(tt.noMatch, pattern)
				if noMatch {
					t.Errorf("%q should NOT match pattern from %q", tt.noMatch, tt.format)
				}
			}
		})
	}
}

func TestCompileBackupPatternEncrypted(t *testing.T) {
	pattern, err := storage.CompileBackupPattern("backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	_, encrypted := storage.MatchBackupPattern("backup-data-2024-01-15T10-30-00.tar.gz.gpg", pattern)
	if !encrypted {
		t.Fatal("should detect .gpg as encrypted")
	}
	_, encrypted = storage.MatchBackupPattern("backup-data-2024-01-15T10-30-00.tar.gz", pattern)
	if encrypted {
		t.Fatal("should not detect non-.gpg as encrypted")
	}
}

func TestCompileBackupPatternWithCapture(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		filename      string
		wantTimestamp string
		wantMatch     bool
	}{
		{
			"data prefix",
			"backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"backup-data-2024-01-15T10-30-00.tar.gz",
			"2024-01-15T10-30-00",
			true,
		},
		{
			"config prefix",
			"backup-config-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"backup-config-2024-01-15T10-30-00.tar.gz",
			"2024-01-15T10-30-00",
			true,
		},
		{
			"encrypted match",
			"backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"backup-data-2024-01-15T10-30-00.tar.gz.gpg",
			"2024-01-15T10-30-00",
			true,
		},
		{
			"no match",
			"backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}",
			"random-file.txt",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := storage.CompileBackupPatternWithCapture(tt.format)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			ts, ok := storage.ExtractTimestamp(tt.filename, pattern)
			if ok != tt.wantMatch {
				t.Fatalf("match: got %v, want %v", ok, tt.wantMatch)
			}
			if ts != tt.wantTimestamp {
				t.Fatalf("timestamp: got %q, want %q", ts, tt.wantTimestamp)
			}
		})
	}
}

func TestMatchBackupFilename(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantMatch bool
		wantGPG   bool
	}{
		{"standard backup", "backup-2024-01-15T10-30-00.tar.gz", true, false},
		{"encrypted backup", "backup-2024-01-15T10-30-00.tar.gz.gpg", true, true},
		{"not a backup", "random-file.txt", false, false},
		{"partial match", "backup-2024-01-15.tar.gz", false, false},
		{"extra prefix", "old-backup-2024-01-15T10-30-00.tar.gz", false, false},
		{"extra suffix", "backup-2024-01-15T10-30-00.tar.gz.bak", false, false},
		{"empty string", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, encrypted := storage.MatchBackupFilename(tt.filename)
			if match != tt.wantMatch {
				t.Errorf("match: got %v, want %v", match, tt.wantMatch)
			}
			if encrypted != tt.wantGPG {
				t.Errorf("encrypted: got %v, want %v", encrypted, tt.wantGPG)
			}
		})
	}
}
