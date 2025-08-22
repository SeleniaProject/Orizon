package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// SymmetricCipher represents a symmetric encryption algorithm.
type SymmetricCipher int

const (
	AES128 SymmetricCipher = iota
	AES192
	AES256
	ChaCha20
	XChaCha20
)

// Cipher represents an encryption/decryption engine.
type Cipher struct {
	algo   SymmetricCipher
	key    []byte
	gcm    cipher.AEAD
	stream cipher.Stream
}

// NewCipher creates a new cipher instance.
func NewCipher(algo SymmetricCipher, key []byte) (*Cipher, error) {
	c := &Cipher{
		algo: algo,
		key:  key,
	}

	switch algo {
	case AES128, AES192, AES256:
		return c.initAES()
	case ChaCha20:
		return nil, errors.New("ChaCha20 not yet implemented")
	case XChaCha20:
		return nil, errors.New("XChaCha20 not yet implemented")
	default:
		return nil, errors.New("unsupported cipher algorithm")
	}
}

func (c *Cipher) initAES() (*Cipher, error) {
	// Validate key length
	switch c.algo {
	case AES128:
		if len(c.key) != 16 {
			return nil, errors.New("AES128 requires 16-byte key")
		}
	case AES192:
		if len(c.key) != 24 {
			return nil, errors.New("AES192 requires 24-byte key")
		}
	case AES256:
		if len(c.key) != 32 {
			return nil, errors.New("AES256 requires 32-byte key")
		}
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	c.gcm = gcm
	return c, nil
}

// Encrypt encrypts plaintext using authenticated encryption.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	if c.gcm == nil {
		return nil, errors.New("cipher not properly initialized")
	}

	// Generate random nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and authenticate
	ciphertext := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext and verifies authentication.
func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if c.gcm == nil {
		return nil, errors.New("cipher not properly initialized")
	}

	nonceSize := c.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify authentication
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptWithKey is a convenience function for one-time encryption.
func EncryptWithKey(plaintext, key []byte, algo SymmetricCipher) ([]byte, error) {
	cipher, err := NewCipher(algo, key)
	if err != nil {
		return nil, err
	}

	return cipher.Encrypt(plaintext)
}

// DecryptWithKey is a convenience function for one-time decryption.
func DecryptWithKey(ciphertext, key []byte, algo SymmetricCipher) ([]byte, error) {
	cipher, err := NewCipher(algo, key)
	if err != nil {
		return nil, err
	}

	return cipher.Decrypt(ciphertext)
}

// StreamCipher represents a stream cipher for continuous encryption.
type StreamCipher struct {
	algo   SymmetricCipher
	key    []byte
	stream cipher.Stream
}

// NewStreamCipher creates a new stream cipher.
func NewStreamCipher(algo SymmetricCipher, key, iv []byte) (*StreamCipher, error) {
	sc := &StreamCipher{
		algo: algo,
		key:  key,
	}

	switch algo {
	case AES128, AES192, AES256:
		return sc.initAESCTR(iv)
	default:
		return nil, errors.New("unsupported stream cipher algorithm")
	}
}

func (sc *StreamCipher) initAESCTR(iv []byte) (*StreamCipher, error) {
	block, err := aes.NewCipher(sc.key)
	if err != nil {
		return nil, err
	}

	if len(iv) != block.BlockSize() {
		return nil, errors.New("IV length must equal block size")
	}

	sc.stream = cipher.NewCTR(block, iv)
	return sc, nil
}

// Encrypt encrypts data using stream cipher.
func (sc *StreamCipher) Encrypt(data []byte) []byte {
	encrypted := make([]byte, len(data))
	sc.stream.XORKeyStream(encrypted, data)
	return encrypted
}

// Decrypt decrypts data using stream cipher.
func (sc *StreamCipher) Decrypt(data []byte) []byte {
	// For CTR mode, encryption and decryption are the same operation
	return sc.Encrypt(data)
}

// Key generation utilities

// GenerateKey generates a random key of specified length.
func GenerateKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	return key, err
}

// GenerateAESKey generates a random AES key.
func GenerateAESKey(bits int) ([]byte, error) {
	switch bits {
	case 128:
		return GenerateKey(16)
	case 192:
		return GenerateKey(24)
	case 256:
		return GenerateKey(32)
	default:
		return nil, errors.New("invalid AES key size")
	}
}

// GenerateIV generates a random initialization vector.
func GenerateIV(length int) ([]byte, error) {
	iv := make([]byte, length)
	_, err := rand.Read(iv)
	return iv, err
}
