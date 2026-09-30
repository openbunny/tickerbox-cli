// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		outNil     bool
		wantErr    bool
		wantStatus int
	}{
		{
			name:   "decodes JSON body",
			status: http.StatusOK,
			body:   `{"ok":true}`,
		},
		{
			name:       "500 returns APIError with status",
			status:     http.StatusInternalServerError,
			body:       "boom",
			wantErr:    true,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:    "empty body with nil out succeeds",
			status:  http.StatusOK,
			body:    "",
			outNil:  true,
			wantErr: false,
		},
		{
			name:       "empty body with non-nil out fails",
			status:     http.StatusOK,
			body:       "",
			wantErr:    true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := New(srv.URL+"/", 0)

			var out map[string]any
			var err error
			if tt.outNil {
				err = c.Get(context.Background(), "thing", nil)
			} else {
				err = c.Get(context.Background(), "thing", &out)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T", err)
				}
				if apiErr.Status != tt.wantStatus {
					t.Errorf("got status %d; want %d", apiErr.Status, tt.wantStatus)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := New(url+"/", 0)
	err := c.Get(context.Background(), "thing", nil)
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Err == nil {
		t.Error("expected wrapped transport error, got nil")
	}
}

func TestPost(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{name: "200 succeeds", status: http.StatusOK},
		{name: "204 succeeds", status: http.StatusNoContent},
		{name: "500 returns APIError", status: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ContentLength != 0 {
					_ = json.NewDecoder(r.Body).Decode(&gotBody)
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			c := New(srv.URL+"/", 0)
			err := c.Post(context.Background(), "thing", map[string]string{"key": "value"})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T", err)
				}
				if apiErr.Status != tt.status {
					t.Errorf("got status %d; want %d", apiErr.Status, tt.status)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotBody["key"] != "value" {
				t.Errorf("got posted body %v; want key=value", gotBody)
			}
		})
	}
}

func TestPostTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := New(url+"/", 0)
	err := c.Post(context.Background(), "thing", nil)
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
}

func TestPostFile(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{name: "200 succeeds", status: http.StatusOK},
		{name: "500 returns APIError", status: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotField string
			var gotContent []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Fatalf("server failed to parse multipart form: %v", err)
				}
				for field, files := range r.MultipartForm.File {
					gotField = field
					f, err := files[0].Open()
					if err != nil {
						t.Fatalf("open uploaded file: %v", err)
					}
					defer func() { _ = f.Close() }()
					buf := make([]byte, files[0].Size)
					_, _ = f.Read(buf)
					gotContent = buf
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			dir := t.TempDir()
			filePath := filepath.Join(dir, "firmware.bin")
			if err := os.WriteFile(filePath, []byte("payload"), 0o644); err != nil {
				t.Fatalf("write test file: %v", err)
			}

			c := New(srv.URL+"/", 0)
			err := c.PostFile(context.Background(), "thing", "file", filePath)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T", err)
				}
				if apiErr.Status != tt.status {
					t.Errorf("got status %d; want %d", apiErr.Status, tt.status)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotField != "file" {
				t.Errorf("got field %q; want %q", gotField, "file")
			}
			if string(gotContent) != "payload" {
				t.Errorf("got content %q; want %q", gotContent, "payload")
			}
		})
	}
}

func TestPostFileMissingFile(t *testing.T) {
	c := New("http://example.invalid/", 0)
	err := c.PostFile(context.Background(), "thing", "file", "/nonexistent/path/firmware.bin")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
}

func TestRetry(t *testing.T) {
	tests := []struct {
		name       string
		failCount  int
		retries    int
		wantErr    bool
		wantCalls  int
		wantStatus int
		op         func(*Client) error
	}{
		{name: "GET succeeds with no failures", failCount: 0, retries: 2, wantCalls: 1, op: func(c *Client) error { return c.Get(context.Background(), "thing", nil) }},
		{name: "GET succeeds after one transient 503", failCount: 1, retries: 2, wantCalls: 2, op: func(c *Client) error { return c.Get(context.Background(), "thing", nil) }},
		{name: "GET exhausts retries into APIError", failCount: 5, retries: 2, wantErr: true, wantCalls: 3, wantStatus: http.StatusServiceUnavailable, op: func(c *Client) error { return c.Get(context.Background(), "thing", nil) }},
		{name: "POST succeeds with no failures", failCount: 0, retries: 2, wantCalls: 1, op: func(c *Client) error { return c.Post(context.Background(), "thing", map[string]string{"key": "value"}) }},
		{name: "POST succeeds after one transient 503", failCount: 1, retries: 2, wantCalls: 2, op: func(c *Client) error { return c.Post(context.Background(), "thing", map[string]string{"key": "value"}) }},
		{name: "POST exhausts retries into APIError", failCount: 5, retries: 2, wantErr: true, wantCalls: 3, op: func(c *Client) error { return c.Post(context.Background(), "thing", map[string]string{"key": "value"}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls <= tt.failCount {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			c := New(srv.URL+"/", 0)
			c.Retries = tt.retries
			c.sleep = func(time.Duration) {}

			err := tt.op(c)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T", err)
				}
				if tt.wantStatus != 0 && apiErr.Status != tt.wantStatus {
					t.Errorf("got status %d; want %d", apiErr.Status, tt.wantStatus)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if calls != tt.wantCalls {
				t.Errorf("got %d calls; want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestRetryDoesNotApplyTo4xx(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL+"/", 0)
	c.Retries = 3
	c.sleep = func(time.Duration) {}

	if err := c.Get(context.Background(), "thing", nil); err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("got %d calls; want 1 (a 4xx is not transient)", calls)
	}
}

func TestRetryTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := New(url+"/", 0)
	c.Retries = 2
	var sleeps int
	c.sleep = func(time.Duration) { sleeps++ }

	err := c.Get(context.Background(), "thing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if sleeps != c.Retries {
		t.Errorf("got %d backoff sleeps; want %d", sleeps, c.Retries)
	}
}

func TestAPIErrorMessages(t *testing.T) {
	withStatus := &APIError{Path: "thing", Status: 500}
	if got := withStatus.Error(); got != "thing: HTTP 500" {
		t.Errorf("got %q; want %q", got, "thing: HTTP 500")
	}

	wrapped := errors.New("connection refused")
	withErr := &APIError{Path: "thing", Err: wrapped}
	if got := withErr.Error(); got != "thing: connection refused" {
		t.Errorf("got %q; want %q", got, "thing: connection refused")
	}
	if !errors.Is(withErr, wrapped) {
		t.Error("expected APIError to unwrap to the wrapped error")
	}
}
