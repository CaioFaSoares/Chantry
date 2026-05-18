package pocketbase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client handles REST HTTP communication with the PocketBase API.
// Thread safety is guaranteed via sync.RWMutex on the admin JWT token.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	adminToken string
	mu         sync.RWMutex
}

// NewClient instantiates a new PocketBase REST Client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// adminAuthRequest defines the request body format for admin password authentication.
type adminAuthRequest struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}

// adminAuthResponse defines the response body containing the JWT token.
type adminAuthResponse struct {
	Token string `json:"token"`
}

// Authenticate performs admin authentication and caches the returned JWT token.
func (c *Client) Authenticate(email, password string) error {
	reqBody := adminAuthRequest{
		Identity: email,
		Password: password,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	url := fmt.Sprintf("%sapi/admins/auth-with-password", c.BaseURL)
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(respBodyBytes))
	}

	var authResp adminAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	if authResp.Token == "" {
		return fmt.Errorf("received empty auth token from PocketBase")
	}

	c.mu.Lock()
	c.adminToken = authResp.Token
	c.mu.Unlock()

	return nil
}

// SendRequest sends an authenticated HTTP request to the PocketBase API.
// It automatically sets "Content-Type: application/json" and "Authorization: Bearer <TOKEN>".
func (c *Client) SendRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.mu.RLock()
	token := c.adminToken
	c.mu.RUnlock()

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}

	return resp, nil
}
