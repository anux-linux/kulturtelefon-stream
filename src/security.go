package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	tokenPrefix         = "k_token:"
	securityKeyFileName = "secret"
)

var secreteKey []byte

var (
	//rightsStreamReader = []string{"get_stream"}
	//rightsStreamEditor = []string{"get_stream", "post_stream"}
	//rightsStreamAdmin  = []string{"get_stream", "post_stream", "delete_stream"}
	//rightsIcecastAdmin = []string{"get_stream", "post_stream", "delete_stream", "get_all_streams"}
	//rightsUser         = []string{"change_password"}
	//rightsUserAdmin    = []string{"create_user", "edit_user", "delete_user"}
	rightsAdmin = []string{"change_password", "create_user", "edit_user", "delete_user", "get_all_streams", "get_stream", "post_stream", "delete_stream"}
)

func setSecretKey(secret string) error {

	// Decode the hex string to bytes
	keyBytes, err := hex.DecodeString(string(secret))
	if err != nil {
		logWithCaller("Failed to decode key: "+err.Error(), FatalLog)
		return err
	}

	secreteKey = keyBytes
	return nil
}

func encryptString(plaintext string) (string, error) {
	// Read the security key

	aes, err := aes.NewCipher(secreteKey)
	if err != nil {
		logWithCaller("Failed to create AES cipher: "+err.Error(), FatalLog)
		return "", err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		logWithCaller("Failed to create GCM: "+err.Error(), FatalLog)
		return "", err
	}

	// We need a 12-byte nonce for GCM (modifiable if you use cipher.NewGCMWithNonceSize())
	// A nonce should always be randomly generated for every encryption.
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		logWithCaller("Failed to generate nonce: "+err.Error(), FatalLog)
		return "", err
	}

	// ciphertext here is actually nonce+ciphertext
	// So that when we decrypt, just knowing the nonce size
	// is enough to separate it from the ciphertext.
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return hex.EncodeToString(ciphertext), nil
}

func decryptString(encrypted string) (string, error) {

	aes, err := aes.NewCipher(secreteKey)
	if err != nil {
		logWithCaller("Failed to create AES cipher: "+err.Error(), FatalLog)
		return "", err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		logWithCaller("Failed to create GCM: "+err.Error(), FatalLog)
		return "", err
	}

	// Since we know the ciphertext is actually nonce+ciphertext
	// And len(nonce) == NonceSize(). We can separate the two.
	encryptedBytes, err := hex.DecodeString(encrypted)
	if err != nil {
		logWithCaller("Failed to decode encrypted string: "+err.Error(), FatalLog)
		return "", err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]

	plaintext, err := gcm.Open(nil, []byte(nonce), ciphertext, nil)
	if err != nil {
		logWithCaller("Failed to decrypt: "+err.Error(), FatalLog)
		return "", err
	}

	// Placeholder for actual decryption logic
	return string(plaintext), nil
}

func createToken(username string, rights []string, exparation time.Time) (string, error) {

	timestamp := time.Now().Format(time.RFC3339Nano)
	if exparation.IsZero() {
		return "", fmt.Errorf("exparation time is zero")
	}
	if exparation.Before(time.Now()) {
		return "", fmt.Errorf("exparation time is in the past")
	}
	if len(rights) == 0 {
		return "", fmt.Errorf("no rights provided")
	}
	if len(username) == 0 {
		return "", fmt.Errorf("no user provided")
	}
	exparationTimestamp := exparation.Format(time.RFC3339Nano)

	securePart := timestamp + "|" + exparationTimestamp + "|" + username + "|" + getApplicationName() + "|" + getVersion()
	for _, right := range rights {
		securePart = securePart + "|" + right

	}

	encryptedSecurePart, err := encryptString(securePart)
	if err != nil {
		logWithCaller("Failed to encrypt token: "+err.Error(), FatalLog)
		return "", err
	}

	// Encode the encrypted secure part to a base64 string
	//encodedSecurePart := hex.EncodeToString([]byte(encryptedSecurePart))

	return tokenPrefix + encryptedSecurePart, nil
}

func checkTokeHasRight(token, right, username string) bool {
	if token == "" {
		logWithCaller("Token is empty", WarnLog)
		return false
	}

	if right == "" {
		logWithCaller("Right is empty", WarnLog)
		return false
	}

	if username == "" {
		logWithCaller("Username is empty", WarnLog)
		return false
	}

	if len(token) < len(tokenPrefix) {
		logWithCaller("Token is too short", WarnLog)
		return false
	}

	if token[:len(tokenPrefix)] != tokenPrefix {
		logWithCaller("Token does not start with k_token: "+token[:len(tokenPrefix)], WarnLog)
		return false
	}

	decrypted, err := decryptString(token[len(tokenPrefix):])
	if err != nil {
		logWithCaller("Failed to decrypt token: "+err.Error(), WarnLog)
		return false
	}

	// Split the decrypted string into parts
	parts := strings.Split(decrypted, "|")
	if len(parts) < 5 {
		logWithCaller("Decrypted token does not have enough parts", WarnLog)
		return false
	}

	timestamp := parts[0]
	if timestamp == "" {
		logWithCaller("Timestamp is empty", WarnLog)
		return false
	}

	timestampTime, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		logWithCaller("Failed to parse timestamp: "+err.Error(), WarnLog)
		return false
	}
	if timestampTime.After(time.Now()) {
		logWithCaller("Creation Timestamp is after today", WarnLog)
		return false
	}

	exparationTimestamp := parts[1]
	if exparationTimestamp == "" {
		logWithCaller("Exparation Timestamp is empty", WarnLog)
		return false
	}
	if len(exparationTimestamp) != len(time.RFC3339Nano) {
		logWithCaller("Exparation Timestamp is not in the correct format", WarnLog)
		return false
	}

	exparationTime, err := time.Parse(time.RFC3339Nano, exparationTimestamp)
	if err != nil {
		logWithCaller("Failed to parse exparation timestamp: "+err.Error(), WarnLog)
		return false
	}
	if exparationTime.Before(time.Now()) {
		logWithCaller("Token expired: "+exparationTimestamp, WarnLog)
		return false
	}
	usernameToken := parts[2]
	if usernameToken == "" {
		logWithCaller("Username is empty", WarnLog)
		return false
	}
	if usernameToken != username {
		logWithCaller("Username does not match", WarnLog)
		return false
	}

	applicationName := parts[3]
	if applicationName != getApplicationName() {
		logWithCaller("Application name does not match", WarnLog)
		return false
	}
	tokenRights := parts[4:]

	for _, r := range tokenRights {
		r = strings.TrimSpace(r)
		right = strings.TrimSpace(right)
		if r == right {
			return true
		}
	}

	logWithCaller("Token does not have the right: "+right, WarnLog)

	return false
}

func getHash(input string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
}

func getHashedPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logWithCaller("Failed to hash password: "+err.Error(), FatalLog)
		return "", err
	}
	return string(bytes), nil
}

func validPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		logWithCaller("Failed to compare password hash: "+err.Error(), InfoLog)
		return false
	}
	return true
}
