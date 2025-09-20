package padd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
)

// EncryptionManager handles age encryption/decryption operations
type EncryptionManager struct {
	recipients []age.Recipient
	identities []age.Identity
	mu         sync.RWMutex
	active     bool
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager() *EncryptionManager {
	return &EncryptionManager{
		recipients: make([]age.Recipient, 0),
		identities: make([]age.Identity, 0),
	}
}

// AddRecipient adds a recipient for encryption (public key)
func (em *EncryptionManager) AddRecipient(publicKey string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	recipient, err := age.ParseX25519Recipient(publicKey)
	if err != nil {
		return fmt.Errorf("failed to parse recipient: %w", err)
	}

	em.recipients = append(em.recipients, recipient)
	return nil
}

// AddRecipientsFromFile loads a set of recipients from a file
// Each line in the file should contain a recipient public key
// Ignores empty lines and lines starting with #
func (em *EncryptionManager) AddRecipientsFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	const fileSizeLimit = 16 << 20  // 16MiB
	const lineLengthLimit = 8 << 10 // 8KiB (same as sshd(8))
	if stat, err := file.Stat(); err == nil && stat.Size() > fileSizeLimit {
		return fmt.Errorf("recipient file size exceeds limit: %d > %d", stat.Size(), fileSizeLimit)
	}

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if len(line) > lineLengthLimit {
			return fmt.Errorf("recipient line exceeds limit: %d > %d", len(line), lineLengthLimit)
		}

		if err := em.AddRecipient(line); err != nil {
			return fmt.Errorf("failed to add recipient: %w", err)
		}
	}

	return nil
}

// AddIdentity adds an identity for decryption (private key)
func (em *EncryptionManager) AddIdentity(identityStr string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	identity, err := age.ParseX25519Identity(identityStr)
	if err != nil {
		return fmt.Errorf("failed to parse identity: %w", err)
	}

	em.identities = append(em.identities, identity)
	return nil
}

// AddIdentitiesFromFile loads an identity from a file
func (em *EncryptionManager) AddIdentitiesFromFile(filePath string) error {
	keyFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open key file: %w", err)
	}

	defer func(keyFile *os.File) {
		_ = keyFile.Close()
	}(keyFile)

	identities, err := age.ParseIdentities(keyFile)
	if err != nil {
		return fmt.Errorf("failed to parse identities: %w", err)
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	for _, identity := range identities {
		em.identities = append(em.identities, identity)
	}

	return nil
}

// Activate starts an encryption session
func (em *EncryptionManager) Activate() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.active = true
}

// Deactivate stops the encryption session
func (em *EncryptionManager) Deactivate() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.active = false
}

// IsActive returns true if encryption is currently active
func (em *EncryptionManager) IsActive() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.active
}

// HasRecipients returns true if any recipients are configured
func (em *EncryptionManager) HasRecipients() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return len(em.recipients) > 0
}

// HasIdentities returns true if any identities are configured
func (em *EncryptionManager) HasIdentities() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return len(em.identities) > 0
}

// Encrypt encrypts content using the configured recipients
func (em *EncryptionManager) Encrypt(content string) ([]byte, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if len(em.recipients) == 0 {
		return nil, fmt.Errorf("no recipients configured for encryption")
	}

	var buf bytes.Buffer

	encryptWriter, err := age.Encrypt(&buf, em.recipients...)
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypt writer: %w", err)
	}

	if _, err := io.WriteString(encryptWriter, content); err != nil {
		_ = encryptWriter.Close()
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	if err := encryptWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encrypt writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decrypt decrypts encrypted content using the configured identities
func (em *EncryptionManager) Decrypt(encryptedContent []byte) (string, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if len(em.identities) == 0 {
		return "", fmt.Errorf("no identities configured for decryption")
	}

	reader := bytes.NewReader(encryptedContent)

	decryptReader, err := age.Decrypt(reader, em.identities...)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, decryptReader); err != nil {
		return "", fmt.Errorf("failed to read decrypted content: %w", err)
	}

	return buf.String(), nil
}

// IsAgeEncrypted checks if content is age encrypted by looking for the format header
func IsAgeEncrypted(content []byte) bool {
	if len(content) < 16 {
		return false
	}

	// Check for the age format header as specified in the format spec
	return bytes.HasPrefix(content, []byte("age-encryption.org/v1"))
}

// HasEncryptedFrontmatter checks if content has encrypted: true in frontmatter
func HasEncryptedFrontmatter(content string) bool {
	lines := SplitLines(content)

	bounds := findFrontmatter(lines)
	if !bounds.Found {
		return false
	}

	// Parse frontmatter lines
	for i := bounds.Start + 1; i < bounds.End; i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "encrypted:") {
			value := strings.TrimPrefix(line, "encrypted:")
			return strings.ToLower(value) == "true" || strings.ToLower(value) == "yes"
		}
	}

	return false
}

// SaveKeyPairToFiles saves a key pair to separate files
func SaveKeyPairToFiles(publicKey, privateKey, keysDir, baseName string) (publicPath, privatePath string, err error) {
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create key directory: %w", err)
	}

	publicPath = filepath.Join(keysDir, baseName+".pub")
	privatePath = filepath.Join(keysDir, baseName+".txt")

	// Save the public key
	if err := os.WriteFile(publicPath, []byte(publicKey+"\n"), 0644); err != nil {
		return "", "", fmt.Errorf("failed to save public key: %w", err)
	}

	// Save the private key with restricted permissions
	privateContent := fmt.Sprintf("# age identity file\n# generated: %s\n%s\n",
		time.Now().Format("2006-01-02 15:04:05"), privateKey)
	if err := os.WriteFile(privatePath, []byte(privateContent), 0600); err != nil {
		return "", "", fmt.Errorf("failed to save private key: %w", err)
	}

	return publicPath, privatePath, nil
}

// GenerateNewEncryptionPair generates a new key pair and saves it to the specified directory
func GenerateNewEncryptionPair(keysDir string) (publicKey, privateKey, publicPath, privatePath string, err error) {
	//em := NewEncryptionManager()
	//publicKey, privateKey, err = em.GenerateKeyPair()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate identity: %w", err)
	}

	privateKey = identity.String()
	publicKey = identity.Recipient().String()

	if strings.TrimSpace(keysDir) == "" {
		return publicKey, privateKey, "", "", fmt.Errorf("keys directory must be specified")
	}

	// Create a default filename with a timestamp
	now := time.Now()
	baseName := fmt.Sprintf("padd-key-%s", now.Format("2006-01-02-15-04-05"))

	publicPath, privatePath, err = SaveKeyPairToFiles(publicKey, privateKey, keysDir, baseName)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to save key pair: %w", err)
	}

	return publicKey, privateKey, publicPath, privatePath, nil
}

// LoadEncryptionKeys loads encryption keys from the specified files.
// identitiesFile: file containing one or more age identities as described in the age spec
// recipientsFile: file containing one or more age recipient public keys
func (em *EncryptionManager) LoadEncryptionKeys(identitiesFile, recipientsFile string) error {
	// If identities is empty, return immediately
	if identitiesFile == "" {
		return fmt.Errorf("no identity file specified")
	}

	if recipientsFile == "" {
		return fmt.Errorf("no recipient file specified")
	}

	// Load identities
	if err := em.AddIdentitiesFromFile(identitiesFile); err != nil {
		return fmt.Errorf("failed to load identity file %s: %w", identitiesFile, err)
	}

	// Load recipients
	if err := em.AddRecipientsFromFile(recipientsFile); err != nil {
		return fmt.Errorf("failed to load recipient file %s: %w", recipientsFile, err)
	}

	em.Activate()
	return nil
}
