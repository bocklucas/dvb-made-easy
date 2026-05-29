package encrypt_test

import (
	"bytes"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/encrypt"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte(`{"projects":[]}`)

	ciphertext, err := encrypt.Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext must differ from plaintext")
	}

	decrypted, err := encrypt.Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	ciphertext, err := encrypt.Encrypt(key1, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = encrypt.Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("same input")

	c1, _ := encrypt.Encrypt(key, plaintext)
	c2, _ := encrypt.Encrypt(key, plaintext)

	if bytes.Equal(c1, c2) {
		t.Fatal("two encryptions of same plaintext must differ (random nonce)")
	}
}

func TestLoadOrCreateKeyCreatesNewKey(t *testing.T) {
	dir := t.TempDir()

	key, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("load or create: %v", err)
	}

	if len(key) != 32 {
		t.Fatalf("key length: got %d, want 32", len(key))
	}
}

func TestLoadOrCreateKeyReusesExistingKey(t *testing.T) {
	dir := t.TempDir()

	key1, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	key2, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Fatal("second call must return same key")
	}
}
