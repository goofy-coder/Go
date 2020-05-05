package main

import (
	"crypto/aes"
	crand "crypto/rand"
	"crypto/sha256"
	"io"
	"testing"
)

func TestEncryptDescrypt(t *testing.T) {
	h := sha256.New()
	h.Write([]byte("my secretes"))
	cryptokey = h.Sum(nil)
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		t.Errorf("iv initialization error: %s", err)
	}

	original := "hello world!"
	ciphertext, err := encrypt([]byte(original), cryptokey, iv)
	if err != nil {
		t.Errorf("Encrytpion error: %s", err)
	}

	decrypted, err := decrypt(ciphertext, cryptokey, iv)
	if err != nil {
		t.Errorf("Decrypt error: %s", err)
	}

	if string(decrypted) != original {
		t.Errorf("%d %d", len(string(decrypted)), len(original))
		t.Errorf("\"%s\" != \"%s\"", string(decrypted), original)
	}
}
