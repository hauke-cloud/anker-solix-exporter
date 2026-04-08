package anker

import (
	"bytes"
	"crypto/ecdh"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	baseURL    = "https://ankerpower-api-eu.anker.com"
	apiVersion = "v2"
)

// Client represents an Anker API client
type Client struct {
	email       string
	password    string
	country     string
	httpClient  *http.Client
	authToken   string
	userID      string
	privateKey  *ecdh.PrivateKey
	sharedKey   []byte
	logger      *zap.Logger
	rateLimiter *RateLimiter
	handler     *RequestHandler
}

// NewClient creates a new Anker API client with a no-op logger
func NewClient(email, password, country string) *Client {
	return NewClientWithLogger(email, password, country, zap.NewNop())
}

// NewClientWithDebug creates a new Anker API client with debug option (deprecated, use NewClientWithLogger)
func NewClientWithDebug(email, password, country string, debug bool) *Client {
	var logger *zap.Logger
	if debug {
		logger, _ = zap.NewDevelopment()
	} else {
		logger = zap.NewNop()
	}
	return NewClientWithLogger(email, password, country, logger)
}

// NewClientWithLogger creates a new Anker API client with a custom logger
func NewClientWithLogger(email, password, country string, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	client := &Client{
		email:       email,
		password:    password,
		country:     country,
		logger:      logger,
		rateLimiter: NewRateLimiter(),
		httpClient: &http.Client{
			Timeout: time.Duration(DefaultRequestTimeout) * time.Second,
		},
	}

	// Initialize request handler
	client.handler = newRequestHandler(client)

	// Generate ECDH key pair for password encryption
	privateKey, sharedKey, err := generateECDHKeys()
	if err != nil {
		logger.Warn("Failed to generate ECDH keys, will use MD5 fallback", zap.Error(err))
		return client
	}

	client.privateKey = privateKey
	client.sharedKey = sharedKey

	return client
}

// SetLogger sets the logger for the client
func (c *Client) SetLogger(logger *zap.Logger) {
	if logger != nil {
		c.logger = logger
	}
}

// SetDebug enables or disables debug logging (deprecated, use SetLogger with appropriate level)
func (c *Client) SetDebug(debug bool) {
	if debug {
		logger, _ := zap.NewDevelopment()
		c.logger = logger
	} else {
		c.logger = zap.NewNop()
	}
}

// IsDebug returns whether debug logging is enabled (deprecated)
func (c *Client) IsDebug() bool {
	return c.logger != zap.NewNop()
}

// debugLog logs a debug message (internal use)
func (c *Client) debugLog(msg string, fields ...zap.Field) {
	c.logger.Debug(msg, fields...)
}

// doRequest performs an HTTP request to the Anker API with rate limiting
func (c *Client) doRequest(method, path string, body []byte, needAuth bool) (*http.Response, error) {
	// Apply rate limiting
	throttleDuration := c.rateLimiter.Wait(path)
	if throttleDuration > 0 {
		c.logger.Warn("Rate limit throttling applied",
			zap.String("endpoint", path),
			zap.Duration("throttle_duration", throttleDuration),
			zap.Int("endpoint_limit", c.rateLimiter.endpointLimit),
		)
	}

	url := baseURL + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Set common headers
	c.setHeaders(req, needAuth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// setHeaders sets the required headers for Anker API requests
func (c *Client) setHeaders(req *http.Request, needAuth bool) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Country", c.country)
	req.Header.Set("Timezone", "GMT+01:00")
	req.Header.Set("Model-Type", "DESKTOP")
	req.Header.Set("App-Name", "anker_power")
	req.Header.Set("Os-Type", "android")
	req.Header.Set("App-Version", "3.0.0")
	req.Header.Set("Language", "en")

	if needAuth && c.authToken != "" {
		req.Header.Set("X-Auth-Token", c.authToken)
		// gtoken is MD5 hash of user_id
		if c.userID != "" {
			gtoken := c.generateGToken()
			req.Header.Set("Gtoken", gtoken)
		}
	}
}

// generateGToken creates a gtoken from the user ID
func (c *Client) generateGToken() string {
	hash := md5.Sum([]byte(c.userID))
	return hex.EncodeToString(hash[:])
}

// Getter methods for testing

// GetSharedKey returns the shared encryption key (for testing)
func (c *Client) GetSharedKey() []byte {
	return c.sharedKey
}

// GetUserID returns the user ID (for testing)
func (c *Client) GetUserID() string {
	return c.userID
}

// GetAuthToken returns the auth token (for testing)
func (c *Client) GetAuthToken() string {
	return c.authToken
}

// SetEndpointLimit sets the maximum requests per endpoint per minute (0 to disable)
func (c *Client) SetEndpointLimit(limit int) {
	c.rateLimiter.SetEndpointLimit(limit)
	c.logger.Info("Rate limit configured",
		zap.Int("endpoint_limit", limit),
		zap.String("unit", "requests/endpoint/minute"),
	)
}

// SetRequestDelay sets the minimum delay between any requests
func (c *Client) SetRequestDelay(delay time.Duration) {
	c.rateLimiter.SetRequestDelay(delay)
	c.logger.Info("Request delay configured",
		zap.Duration("delay", delay),
	)
}

// GetRateLimitStats returns statistics about current rate limiting
func (c *Client) GetRateLimitStats() map[string]interface{} {
	return c.rateLimiter.GetStats()
}
