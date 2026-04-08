package anker

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	// Anker API server's public key (uncompressed format: 04 + x-coordinate + y-coordinate)
	ankerPublicKeyHex = "04c5c00c4f8d1197cc7c3167c52bf7acb054d722f0ef08dcd7e0883236e0d72a3868d9750cb47fa4619248f3d83f0f662671dadc6e2d31c2f41db0161651c7c076"
)

// generateECDHKeys generates ECDH key pair and computes shared secret with Anker's server
func generateECDHKeys() (*ecdh.PrivateKey, []byte, error) {
	curve := ecdh.P256()
	
	// Generate our private key
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDH key: %w", err)
	}

	// Decode Anker's server public key
	serverPubKeyBytes, err := hex.DecodeString(ankerPublicKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode server public key: %w", err)
	}

	// Import server's public key
	serverPubKey, err := curve.NewPublicKey(serverPubKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to import server public key: %w", err)
	}

	// Compute shared secret using ECDH
	sharedSecret, err := privateKey.ECDH(serverPubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	return privateKey, sharedSecret, nil
}

// encryptPassword encrypts the password using AES-256-CBC with the shared secret
func encryptPassword(password string, sharedKey []byte) (string, error) {
	if sharedKey == nil {
		// Fallback to MD5 hash if encryption not available
		return hashPassword(password), nil
	}

	// Use AES-256-CBC with PKCS7 padding
	block, err := aes.NewCipher(sharedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Apply PKCS7 padding
	paddedPassword := pkcs7Pad([]byte(password), aes.BlockSize)

	// Use first 16 bytes of shared key as IV
	iv := sharedKey[:aes.BlockSize]

	// Encrypt
	ciphertext := make([]byte, len(paddedPassword))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPassword)

	// Return base64 encoded ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// hashPassword creates MD5 hash of the password
func hashPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

// pkcs7Pad applies PKCS7 padding to the data
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}
