package navcore

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SignIn verifies the new login before atomically replacing the device configuration.
func SignIn(ctx context.Context, path, server, username string, password []byte, bitrate int) error {
	if len(password) == 0 {
		return errors.New("Enter your password")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("Could not generate a sign-in salt")
	}
	salt := hex.EncodeToString(nonce[:])
	hash := md5.New() // Subsonic token authentication requires MD5(password + salt).
	hash.Write(password)
	hash.Write([]byte(salt))
	cfg := Config{
		Server: strings.TrimRight(strings.TrimSpace(server), "/"), Username: strings.TrimSpace(username),
		Salt: salt, Token: hex.EncodeToString(hash.Sum(nil)), MaxBitrate: bitrate,
	}
	client, err := New(cfg)
	if err != nil {
		return err
	}
	defer client.HTTP.CloseIdleConnections()
	if err = client.Ping(ctx); err != nil {
		return errors.New("Sign-in failed. Check Wi-Fi, server address and login.")
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("Cannot open the SD card data folder: %w", err)
	}
	file, err := os.CreateTemp(dir, ".signin-*")
	if err != nil {
		return fmt.Errorf("Cannot create the SD card configuration: %w", err)
	}
	defer os.Remove(file.Name())
	if err = json.NewEncoder(file).Encode(cfg); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("Could not save the connection to the SD card: %w", err)
	}
	return nil
}
