package anker

import (
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestClientInitialization(t *testing.T) {
	client := NewClient("test@example.com", "testpassword", "DE")

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.sharedKey == nil {
		t.Fatal("Shared key should be generated")
	}

	if len(client.sharedKey) != 32 {
		t.Fatalf("Shared key should be 32 bytes, got %d", len(client.sharedKey))
	}

	if client.privateKey == nil {
		t.Fatal("Private key should be generated")
	}
}

func TestPasswordEncryption(t *testing.T) {
	client := NewClient("test@example.com", "testpassword", "DE")

	encrypted, err := encryptPassword("testpassword", client.sharedKey)
	if err != nil {
		t.Fatalf("Password encryption failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Encrypted password should not be empty")
	}

	// Base64 encoded strings should be longer than plaintext
	if len(encrypted) < len("testpassword") {
		t.Fatal("Encrypted password seems too short")
	}
}

func TestLoginRequestFormat(t *testing.T) {
	client := NewClient("test@example.com", "testpassword", "DE")

	encryptedPassword, _ := encryptPassword("testpassword", client.sharedKey)
	publicKeyBytes := client.privateKey.PublicKey().Bytes()

	reqBody := LoginRequest{
		AB:    "DE",
		Email: "test@example.com",
		ClientSecretInfo: map[string]interface{}{
			"public_key": hex.EncodeToString(publicKeyBytes),
		},
		Enc:         0,
		Password:    encryptedPassword,
		TimeZone:    3600000, // 1 hour
		Transaction: "1234567890123",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal login request: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Failed to parse marshaled JSON: %v", err)
	}

	// Verify all required fields are present
	required := []string{"ab", "email", "client_secret_info", "enc", "password", "time_zone", "transaction"}
	for _, field := range required {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify client_secret_info has public_key
	if secretInfo, ok := parsed["client_secret_info"].(map[string]interface{}); ok {
		if _, ok := secretInfo["public_key"]; !ok {
			t.Error("Missing public_key in client_secret_info")
		}
	} else {
		t.Error("client_secret_info is not a map")
	}
}
