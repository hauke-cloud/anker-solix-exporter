package anker

import (
	"encoding/hex"
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

	var loginResp LoginResponse
	if err := c.handler.execute("POST", EndpointLogin, reqBody, &loginResp, false); err != nil {
		return fmt.Errorf("login failed: %w", err)
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
