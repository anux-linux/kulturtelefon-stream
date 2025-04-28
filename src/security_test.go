package main

import (
	"log"
	"testing"
	"time"
)

func initTest(t *testing.T) {
	setLogLevel(DebugLog)
	// Define a temporary directory for the test
	err := setSecretKey("882093050f95bfb1d2b83510d90393b623f86be241169d5db3ea76d715628ef9")
	if err != nil {
		t.Fatalf("Failed to decode key: %v", err)
	}
}

func TestEncryptDecryptString(t *testing.T) {

	initTest(t)

	// Test data
	plaintext := "MyRandom Token!@#$%^&*()_+ 12 asadfwe 13.12.2034 12:00:00 UTC |asdf_des|asdaf"

	// Encrypt the string
	encrypted, err := encryptString(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt string: %v", err)
	}

	// Decrypt the string
	decrypted, err := decryptString(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt string: %v", err)
	}

	// Check if the decrypted string matches the original plaintext
	if decrypted != plaintext {
		t.Fatalf("Decrypted string does not match plaintext. Got: %s, Expected: %s", decrypted, plaintext)
	}
}

func TestCreateToken(t *testing.T) {
	initTest(t)

	// Test data
	rights := []string{"create_stream", "get_stream", "admin"}

	experationTimestamp := time.Now().Add(24 * 365 * time.Hour)

	// Create a token
	token, err := createToken("test_user", rights, experationTimestamp)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	log.Printf("Created token: %s", token)

	// Check if the token starts with the expected prefix
	expectedPrefix := "k_token:"
	if len(token) <= len(expectedPrefix) || token[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("Token does not start with the expected prefix. Got: %s, Expected prefix: %s", token, expectedPrefix)
	}
}

func TestCreatedTokenDublicate(t *testing.T) {
	initTest(t)

	// Test data
	rights := []string{"create_stream", "get_stream", "admin"}

	experationTimestamp := time.Now().Add(24 * 365 * time.Hour)

	// Create a token
	token1, err := createToken("test_user", rights, experationTimestamp)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Create a secound token
	token2, err := createToken("test_user", rights, experationTimestamp)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	log.Printf("Created 2 tokens equal: \n%s \n%s", token1, token2)

	//Check if the token are equal
	if len(token1) != len(token2) && token1 == token2 {
		t.Fatalf("Token are equal: \n%s \n%s", token1, token2)
	}
}

func TestCreatedTokenContent(t *testing.T) {
	initTest(t)

	// Test data
	rights := []string{"create_stream", "get_stream", "admin"}

	experationTimestamp := time.Now().Add(24 * 365 * time.Hour)

	// Create a secound token
	token, err := createToken("test_user", rights, experationTimestamp)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Check if the token starts with the expected prefix
	expectedPrefix := "k_token:"

	// Decrypt the token
	decrypted, err := decryptString(token[len(expectedPrefix):])
	if err != nil {
		t.Fatalf("Failed to decrypt token: %v", err)
	}

	log.Printf("Decrypted token: %s", decrypted)
}

func TestCheckTokenRights(t *testing.T) {
	initTest(t)

	// Test data
	rights := []string{"create_stream", "get_stream", "admin"}

	experationTimestamp := time.Now().Add(2 * time.Hour)

	// Create a secound token
	token, err := createToken("test_user", rights, experationTimestamp)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Check if the token has the right
	if !checkTokeHasRight(token, "create_stream", "test_user") {
		t.Fatalf("Token does not have the right")
	}

	if checkTokeHasRight(token, "delete_stream", "test_user") {
		t.Fatalf("Token does not have the right")
	}

	if checkTokeHasRight(token, "create_stream", "wrong_user") {
		t.Fatalf("Token is for the wrong user")
	}

	fastExperationTimestamp := time.Now().Add(1 * time.Second)

	// Create a secound token
	expiredToken, err := createToken("fast_user", rights, fastExperationTimestamp)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	time.Sleep(2 * time.Second)

	if checkTokeHasRight(expiredToken, "create_stream", "fast_user") {
		t.Fatalf("Token is expired")
	}
}
