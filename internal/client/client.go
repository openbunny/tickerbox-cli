// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultHost = "http://tickerbox.local"

const (
	defaultTimeout = 10 * time.Second
	retryBackoff   = 200 * time.Millisecond
)

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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func Make(host string) *Client {
	base := strings.TrimRight(firstNonEmpty(host, os.Getenv("TICKERBOX_HOST"), DefaultHost), "/") + "/rest/"
	return New(base, defaultTimeout)
}

func (c *Client) url(path string) string {
	return c.base + strings.TrimLeft(path, "/")
}

func (c *Client) Get(path string, out any) error {
	body, status, err := c.GetRaw(path)
	if err != nil {
		return &APIError{Path: path, Err: err}
	}
	if status >= 400 {
		return &APIError{Path: path, Status: status}
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return &APIError{Path: path, Status: status, Err: fmt.Errorf("decode response: %w", err)}
		}
	}
	return nil
}

func (c *Client) GetRaw(path string) ([]byte, int, error) {
	var body []byte
	var status int
	var err error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		body, status, err = c.getOnce(path)
		if !shouldRetry(status, err) {
			break
		}
		if attempt < c.Retries {
			c.sleep(retryBackoff)
		}
	}
	return body, status, err
}

func (c *Client) getOnce(path string) ([]byte, int, error) {
	resp, err := c.http.Get(c.url(path))
	if err != nil {
		return nil, 0, fmt.Errorf("request %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s: %w", path, err)
	}
	return body, resp.StatusCode, nil
}

func (c *Client) Post(path string, body any) error {
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
		status, err = c.postOnce(path, encoded)
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

func (c *Client) postOnce(path string, encoded []byte) (int, error) {
	var reader io.Reader
	if encoded != nil {
		reader = bytes.NewReader(encoded)
	}

	resp, err := c.http.Post(c.url(path), "application/json", reader)
	if err != nil {
		return 0, fmt.Errorf("request %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode, nil
}

func (c *Client) PostFile(path, field, filePath string) error {
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

	req, err := http.NewRequest(http.MethodPost, c.url(path), &buf)
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
