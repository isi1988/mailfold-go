// Package mailfold is the official Go client SDK for Mailfold
// (https://github.com/isi1988/Mailfold), a self-hosted webmail/admin
// backend. It wraps Mailfold's per-mailbox REST API using only the
// standard library (net/http, encoding/json) — no third-party
// dependencies.
package mailfold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a Mailfold API client bound to a single base URL and API key.
// Construct one with New and reuse it; it is safe for concurrent use.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a Client for the given Mailfold instance base URL (e.g.
// "https://real.mailfold.site" or a self-hosted URL) and API key
// (a "mf_live_..." bearer token). The token is sent verbatim and is never
// parsed or validated client-side.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// WithHTTPClient overrides the underlying *http.Client (e.g. to set a
// custom timeout or transport). Returns the same Client for chaining.
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	c.httpClient = hc
	return c
}

// Send sends an email from the mailbox the API key is bound to.
// Requires the mail:send scope.
func (c *Client) Send(req SendRequest) (*SendResponse, error) {
	var out SendResponse
	if err := c.do(http.MethodPost, "/api/v1/mail/send", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Folders lists the mailbox's IMAP folders. Requires the mail:read scope.
func (c *Client) Folders() ([]Folder, error) {
	var out []Folder
	if err := c.do(http.MethodGet, "/api/v1/mail/folders", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MessagesOptions configures Client.Messages. Zero values mean "use the
// server default" (folder defaults to INBOX, limit defaults to 50).
type MessagesOptions struct {
	Folder string
	Limit  int
}

// Messages lists message headers in a folder, newest first. Requires the
// mail:read scope.
func (c *Client) Messages(opts MessagesOptions) ([]MessageHeader, error) {
	q := url.Values{}
	if opts.Folder != "" {
		q.Set("folder", opts.Folder)
	}
	if opts.Limit != 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	var out []MessageHeader
	if err := c.do(http.MethodGet, "/api/v1/mail/messages", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Message fetches a single full message (body + attachment metadata) by
// folder and UID. Requires the mail:read scope.
func (c *Client) Message(folder string, uid int) (*Message, error) {
	q := url.Values{}
	if folder != "" {
		q.Set("folder", folder)
	}
	q.Set("uid", strconv.Itoa(uid))
	var out Message
	if err := c.do(http.MethodGet, "/api/v1/mail/message", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteMessage deletes (or marks \Deleted, per server semantics) the
// message at folder/uid. Requires the mail:write scope.
func (c *Client) DeleteMessage(folder string, uid int) error {
	body := struct {
		Folder string `json:"folder"`
		UID    int    `json:"uid"`
	}{folder, uid}
	var out StatusResponse
	return c.do(http.MethodDelete, "/api/v1/mail/message", nil, body, &out)
}

// Search searches a folder for messages matching q. Requires the mail:read
// scope.
func (c *Client) Search(folder, q string) ([]MessageHeader, error) {
	vals := url.Values{}
	if folder != "" {
		vals.Set("folder", folder)
	}
	vals.Set("q", q)
	var out []MessageHeader
	if err := c.do(http.MethodGet, "/api/v1/mail/search", vals, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Attachment downloads a single attachment's raw bytes by folder, message
// UID, and attachment index. Requires the mail:read scope.
//
// Unlike every other endpoint, this one returns raw binary bytes rather
// than a JSON envelope, so it is handled outside of Client.do.
func (c *Client) Attachment(folder string, uid int, index int) (*AttachmentData, error) {
	q := url.Values{}
	if folder != "" {
		q.Set("folder", folder)
	}
	q.Set("uid", strconv.Itoa(uid))
	q.Set("index", strconv.Itoa(index))

	httpReq, err := c.newRequest(http.MethodGet, "/api/v1/mail/attachment", q, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mailfold: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mailfold: reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp, data)
	}

	filename := ""
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			filename = params["filename"]
		}
	}
	return &AttachmentData{
		Data:        data,
		ContentType: resp.Header.Get("Content-Type"),
		Filename:    filename,
	}, nil
}

// SetFlagRequest is the payload for Client.SetFlag.
type SetFlagRequest struct {
	Folder string `json:"folder"`
	UID    int    `json:"uid"`
	Flag   Flag   `json:"flag"`
	Set    bool   `json:"set"`
}

// SetFlag sets or clears a single IMAP flag on a message. Requires the
// mail:write scope.
func (c *Client) SetFlag(req SetFlagRequest) error {
	var out StatusResponse
	return c.do(http.MethodPost, "/api/v1/mail/flag", nil, req, &out)
}

func (c *Client) newRequest(method, path string, query url.Values, body interface{}) (*http.Request, error) {
	full := c.baseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("mailfold: encoding request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, full, reader)
	if err != nil {
		return nil, fmt.Errorf("mailfold: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) do(method, path string, query url.Values, body interface{}, out interface{}) error {
	req, err := c.newRequest(method, path, query, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mailfold: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("mailfold: reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp, data)
	}

	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("mailfold: decoding response body: %w", err)
	}
	return nil
}

func parseAPIError(resp *http.Response, data []byte) error {
	apiErr := &APIError{StatusCode: resp.StatusCode, Message: "unknown error"}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &body); err == nil && body.Error != "" {
		apiErr.Message = body.Error
	} else if len(data) > 0 {
		apiErr.Message = string(data)
	}

	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			apiErr.RetryAfter = secs
			apiErr.HasRetryAfter = true
		}
	}
	return apiErr
}
