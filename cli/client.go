// Package cli is the tctl client over the Tennon API. It never touches the
// store directly, so validation, policy, and audit apply uniformly. It depends
// only on domain (for the object shapes) and the HTTP API.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is a thin HTTP client for the Tennon API.
type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

// NewClient builds an API client for baseURL, sending token as a bearer.
func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, Token: token, http: &http.Client{}}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		if e.Error != "" {
			return fmt.Errorf("%s %s: %s", method, path, e.Error)
		}
		return fmt.Errorf("%s %s: %s", method, path, resp.Status)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
