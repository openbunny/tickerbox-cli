// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultHost = "http://tickerbox.local"

const retryBackoff = 200 * time.Millisecond

type APIError struct {
	Path   string
	Status int
	Err    error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("%s: HTTP %d", e.Path, e.Status)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

type Client struct {
	http    *http.Client
	base    string
	Retries int
	sleep   func(time.Duration)
}

func New(base string, timeout time.Duration) *Client {
	return &Client{
		http:  &http.Client{Timeout: timeout},
		base:  base,
		sleep: time.Sleep,
	}
}

func shouldRetry(status int, err error) bool {
	return err != nil || status >= http.StatusInternalServerError
}

func (c *Client) url(path string) string {
	return c.base + strings.TrimLeft(path, "/")
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.get(ctx, path, path, out)
}

// GetLabeled behaves like Get, but reports label in any APIError instead of path —
// for a path carrying a secret (e.g. an API key in a query string) that must never
// reach a formatted or printed error.
func (c *Client) GetLabeled(ctx context.Context, path, label string, out any) error {
	return c.get(ctx, path, label, out)
}

func (c *Client) get(ctx context.Context, path, label string, out any) error {
	body, status, err := c.getRaw(ctx, path, label)
	if err != nil {
		return &APIError{Path: label, Err: err}
	}
	if status >= 400 {
		return &APIError{Path: label, Status: status}
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return &APIError{Path: label, Status: status, Err: fmt.Errorf("decode response: %w", err)}
		}
	}
	return nil
}

func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, int, error) {
	return c.getRaw(ctx, path, path)
}

// getRaw issues the request against path but names label, not path, in any
// wrapped error — path is what a labeled caller (e.g. FMPClient) must keep out
// of formatted or printed text, such as an API key carried in a query string.
func (c *Client) getRaw(ctx context.Context, path, label string) ([]byte, int, error) {
	var body []byte
	var status int
	var err error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		body, status, err = c.getOnce(ctx, path, label)
		if !shouldRetry(status, err) {
			break
		}
		if attempt < c.Retries {
			c.sleep(retryBackoff)
		}
	}
	return body, status, err
}

// redactRequestError replaces a *url.Error's URL — net/http embeds the full
// request URL, query string included, in both a request-build failure and a
// Do failure (transport error, timeout, TLS failure) — with safeURL, so a
// secret in the query string (e.g. an FMP API key) never reaches the error's
// formatted text.
func redactRequestError(err error, safeURL string) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		redacted := *uerr
		redacted.URL = safeURL
		return &redacted
	}
	return err
}

func (c *Client) getOnce(ctx context.Context, path, label string) ([]byte, int, error) {
	safeURL := c.url(label)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request %s: %w", label, redactRequestError(err, safeURL))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request %s: %w", label, redactRequestError(err, safeURL))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s: %w", label, err)
	}
	return body, resp.StatusCode, nil
}

func (c *Client) Post(ctx context.Context, path string, body any) error {
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return &APIError{Path: path, Err: fmt.Errorf("encode request: %w", err)}
		}
	}

	var status int
	var err error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		status, err = c.postOnce(ctx, path, encoded)
		if !shouldRetry(status, err) {
			break
		}
		if attempt < c.Retries {
			c.sleep(retryBackoff)
		}
	}

	if err != nil {
		return &APIError{Path: path, Err: err}
	}
	if status >= 400 {
		return &APIError{Path: path, Status: status}
	}
	return nil
}

func (c *Client) postOnce(ctx context.Context, path string, encoded []byte) (int, error) {
	var reader io.Reader
	if encoded != nil {
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), reader)
	if err != nil {
		return 0, fmt.Errorf("build request %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode, nil
}

func (c *Client) PostFile(ctx context.Context, path, field, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return &APIError{Path: path, Err: fmt.Errorf("open %s: %w", filePath, err)}
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(field, filepath.Base(filePath))
	if err != nil {
		return &APIError{Path: path, Err: fmt.Errorf("build multipart form: %w", err)}
	}
	if _, err := io.Copy(part, file); err != nil {
		return &APIError{Path: path, Err: fmt.Errorf("read %s: %w", filePath, err)}
	}
	if err := writer.Close(); err != nil {
		return &APIError{Path: path, Err: fmt.Errorf("close multipart form: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), &buf)
	if err != nil {
		return &APIError{Path: path, Err: fmt.Errorf("build request: %w", err)}
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return &APIError{Path: path, Err: fmt.Errorf("request %s: %w", path, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return &APIError{Path: path, Status: resp.StatusCode}
	}
	return nil
}
