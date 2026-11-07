package navcore

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSignInVerifiesBeforeReplacingConnection(t *testing.T) {
	const password = "sample-Pass+/9"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/music/rest/ping.view" || q.Get("u") != "tester" || len(q.Get("s")) != 32 {
			t.Error("unexpected sign-in request")
		}
		if q.Has("password") || q.Has("p") || bytes.Contains([]byte(r.URL.String()), []byte(password)) {
			t.Error("received a plaintext password")
		}
		token := fmt.Sprintf("%x", md5.Sum([]byte(password+q.Get("s"))))
		if q.Get("t") != token {
			fmt.Fprint(w, `{"subsonic-response":{"status":"failed","error":{"code":40}}}`)
			return
		}
		fmt.Fprint(w, `{"subsonic-response":{"status":"ok"}}`)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "data", "config.json")
	login := func(pass string) error {
		return SignIn(context.Background(), path, server.URL+"/music/", "tester", []byte(pass), 128)
	}
	if err := login(password); err != nil {
		t.Fatal(err)
	}
	first, err := ReadConfig(path)
	if err != nil || first.MaxBitrate != 128 || first.Server != server.URL+"/music" {
		t.Fatal("saved config mismatch", err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || bytes.Contains(saved, []byte(password)) {
		t.Fatal("invalid credential storage", err)
	}
	if err := login("wrong"); err == nil {
		t.Fatal("invalid login accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, after) {
		t.Fatal("failed login changed saved config")
	}
	if err := login(password); err != nil {
		t.Fatal(err)
	}
	second, err := ReadConfig(path)
	if err != nil || first.Salt == second.Salt || first.Token == second.Token {
		t.Fatal("expected fresh credentials")
	}
}
