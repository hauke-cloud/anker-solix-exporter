package anker

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Login authenticates with the Anker API and obtains an auth token
func (c *Client) Login() error {
	c.debugLog("Attempting login", zap.String("email", c.email))

	// Encrypt password using AES-256-CBC with the shared secret
	encryptedPassword, err := encryptPassword(c.password, c.sharedKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Get current timezone offset in milliseconds
	now := time.Now()
	_, offset := now.Zone()
	timezoneOffset := int64(offset * 1000) // Convert seconds to milliseconds

	// Generate transaction ID (Unix timestamp in milliseconds)
	transaction := fmt.Sprintf("%d", now.UnixMilli())

	// Get client public key in hex format
	publicKeyBytes := c.privateKey.PublicKey().Bytes()

	reqBody := LoginRequest{
		AB:    c.country,
		Email: c.email,
		ClientSecretInfo: map[string]interface{}{
			"public_key": hex.EncodeToString(publicKeyBytes),
		},
		Enc:         0,
		Password:    encryptedPassword,
		TimeZone:    timezoneOffset,
		Transaction: transaction,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := c.doRequest("POST", EndpointLogin, body, false)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResp.Code != 0 {
		return fmt.Errorf("login failed: %s (code: %d)", loginResp.Msg, loginResp.Code)
	}

	c.authToken = loginResp.Data.AuthToken
	c.userID = loginResp.Data.UserID

	c.logger.Info("Login successful", 
		zap.String("user_id", c.userID),
		zap.String("email", c.email),
	)

	return nil
}

// IsAuthenticated returns whether the client has a valid auth token
func (c *Client) IsAuthenticated() bool {
	return c.authToken != ""
}
