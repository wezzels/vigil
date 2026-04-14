package c2bmc

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client implements C2BMCClient using REST API
type Client struct {
	config     *C2BMCConfig
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new C2BMC client
func NewClient(config *C2BMCConfig) (*Client, error) {
	if config == nil {
		config = DefaultC2BMCConfig()
	}

	// Create HTTP transport
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		TLSClientConfig:     &tls.Config{},
	}

	// Configure TLS
	if config.EnableMTLS {
		if config.CertFile == "" || config.KeyFile == "" {
			return nil, fmt.Errorf("mTLS enabled but cert/key files not specified")
		}

		// Load client certificate
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}

		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}

		// Load CA certificate
		if config.CAFile != "" {
			caCert, err := os.ReadFile(config.CAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}

			transport.TLSClientConfig.RootCAs = caCertPool
		}
	}

	transport.TLSClientConfig.InsecureSkipVerify = config.InsecureSkipVerify

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		baseURL:    config.Endpoint,
	}, nil
}

// SubmitAlert submits an alert to C2BMC
func (c *Client) SubmitAlert(ctx context.Context, req *AlertRequest) (*AlertResponse, error) {
	// Set timestamp if not set
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	// Generate alert ID if not set
	if req.AlertID == "" {
		req.AlertID = generateAlertID()
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/alerts", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var alertResp AlertResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &alertResp, nil
}

// GetAlertStatus gets the status of an alert
func (c *Client) GetAlertStatus(ctx context.Context, alertID string) (*AlertResponse, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/alerts/%s", alertID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var alertResp AlertResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &alertResp, nil
}

// CancelAlert cancels an alert
func (c *Client) CancelAlert(ctx context.Context, alertID string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/alerts/%s", alertID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// SubmitTrack submits a track to C2BMC
func (c *Client) SubmitTrack(ctx context.Context, track *TrackData) error {
	// Set timestamps if not set
	if track.FirstDetect.IsZero() {
		track.FirstDetect = time.Now()
	}
	if track.LastUpdate.IsZero() {
		track.LastUpdate = time.Now()
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/tracks", track)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetTrack gets a track from C2BMC
func (c *Client) GetTrack(ctx context.Context, trackID string) (*TrackData, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/tracks/%s", trackID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var track TrackData
	if err := json.NewDecoder(resp.Body).Decode(&track); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &track, nil
}

// CorrelateTracks correlates multiple tracks
func (c *Client) CorrelateTracks(ctx context.Context, req *TrackCorrelationRequest) (*TrackCorrelationResponse, error) {
	// Set timestamp if not set
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/tracks/correlate", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var corrResp TrackCorrelationResponse
	if err := json.NewDecoder(resp.Body).Decode(&corrResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &corrResp, nil
}

// HealthCheck performs a health check
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetStatus gets the system status
func (c *Client) GetStatus(ctx context.Context) (*SystemStatus, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status SystemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		resp, err := c.doSingleRequest(ctx, method, path, body)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if c2bmcErr, ok := err.(*C2BMCError); ok {
			if !c2bmcErr.IsRetryable() {
				return nil, err
			}
		}

		// Wait before retry
		if attempt < c.config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay):
			}
		}
	}

	return nil, lastErr
}

// doSingleRequest performs a single HTTP request
func (c *Client) doSingleRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &C2BMCError{
			Code:    0,
			Message: "Request failed",
			Detail:  err.Error(),
		}
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		return nil, &C2BMCError{
			Code:    resp.StatusCode,
			Message: http.StatusText(resp.StatusCode),
			Detail:  string(bodyBytes),
		}
	}

	return resp, nil
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("ALERT-%d", time.Now().UnixNano())
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}