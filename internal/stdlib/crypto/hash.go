// Package crypto provides cryptographic primitives and algorithms.
// This package implements hash functions, encryption, digital signatures,
// and key management without external dependencies.
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	"math/big"
)

// HashAlgorithm represents supported hash algorithms.
type HashAlgorithm int

const (
	SHA256 HashAlgorithm = iota
	SHA512
	CRC32
	CRC64
	Blake2b
	Blake3
)

// Hash represents a cryptographic hash function.
type Hash struct {
	algo HashAlgorithm
	h    hash.Hash
}

// NewHash creates a new hash instance for the specified algorithm.
func NewHash(algo HashAlgorithm) (*Hash, error) {
	var h hash.Hash

	switch algo {
	case SHA256:
		h = sha256.New()
	case SHA512:
		h = sha512.New()
	case CRC32:
		h = crc32.NewIEEE()
	case CRC64:
		h = crc64.New(crc64.MakeTable(crc64.ECMA))
	case Blake2b:
		return nil, fmt.Errorf("Blake2b not yet implemented")
	case Blake3:
		return nil, fmt.Errorf("Blake3 not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %d", algo)
	}

	return &Hash{
		algo: algo,
		h:    h,
	}, nil
}

// Write adds data to the hash.
func (h *Hash) Write(data []byte) (int, error) {
	return h.h.Write(data)
}

// Sum returns the hash sum.
func (h *Hash) Sum() []byte {
	return h.h.Sum(nil)
}

// SumHex returns the hash sum as a hexadecimal string.
func (h *Hash) SumHex() string {
	return hex.EncodeToString(h.Sum())
}

// Reset resets the hash to its initial state.
func (h *Hash) Reset() {
	h.h.Reset()
}

// Convenience functions for common hash operations.

// SHA256Sum computes SHA256 hash of data.
func SHA256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

// SHA256SumHex computes SHA256 hash of data and returns as hex string.
func SHA256SumHex(data []byte) string {
	return hex.EncodeToString(SHA256Sum(data))
}

// SHA512Sum computes SHA512 hash of data.
func SHA512Sum(data []byte) []byte {
	sum := sha512.Sum512(data)
	return sum[:]
}

// SHA512SumHex computes SHA512 hash of data and returns as hex string.
func SHA512SumHex(data []byte) string {
	return hex.EncodeToString(SHA512Sum(data))
}

// CRC32Sum computes CRC32 checksum of data.
func CRC32Sum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// CRC64Sum computes CRC64 checksum of data.
func CRC64Sum(data []byte) uint64 {
	table := crc64.MakeTable(crc64.ECMA)
	return crc64.Checksum(data, table)
}

// HMAC provides Hash-based Message Authentication Code.
type HMAC struct {
	key  []byte
	algo HashAlgorithm
}

// NewHMAC creates a new HMAC instance.
func NewHMAC(key []byte, algo HashAlgorithm) *HMAC {
	return &HMAC{
		key:  key,
		algo: algo,
	}
}

// Sign computes HMAC signature for data.
func (hmac *HMAC) Sign(data []byte) ([]byte, error) {
	h, err := NewHash(hmac.algo)
	if err != nil {
		return nil, err
	}

	// Simplified HMAC implementation
	// In a real implementation, this would follow RFC 2104
	h.Write(hmac.key)
	h.Write(data)
	return h.Sum(), nil
}

// Verify verifies HMAC signature.
func (hmac *HMAC) Verify(data, signature []byte) (bool, error) {
	computed, err := hmac.Sign(data)
	if err != nil {
		return false, err
	}

	if len(computed) != len(signature) {
		return false, nil
	}

	// Constant time comparison
	var result byte
	for i := range computed {
		result |= computed[i] ^ signature[i]
	}

	return result == 0, nil
}

// Random number generation
type Random struct {
	source io.Reader
}

// NewRandom creates a new random number generator.
func NewRandom() *Random {
	return &Random{
		source: rand.Reader,
	}
}

// Bytes generates random bytes.
func (r *Random) Bytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r.source, buf)
	return buf, err
}

// Int generates a random integer in range [0, max).
func (r *Random) Int(max *big.Int) (*big.Int, error) {
	return rand.Int(r.source, max)
}

// Key derivation function (simplified PBKDF2)
type KDF struct {
	iterations int
	keyLen     int
}

// NewKDF creates a new key derivation function.
func NewKDF(iterations, keyLen int) *KDF {
	return &KDF{
		iterations: iterations,
		keyLen:     keyLen,
	}
}

// DeriveKey derives a key from password and salt.
func (kdf *KDF) DeriveKey(password, salt []byte) ([]byte, error) {
	// Simplified PBKDF2 implementation
	// In a real implementation, this would be more sophisticated
	h := sha256.New()
	h.Write(password)
	h.Write(salt)

	key := h.Sum(nil)

	// Perform iterations
	for i := 1; i < kdf.iterations; i++ {
		h.Reset()
		h.Write(key)
		key = h.Sum(nil)
	}

	// Truncate or pad to desired length
	if len(key) > kdf.keyLen {
		return key[:kdf.keyLen], nil
	}

	if len(key) < kdf.keyLen {
		padded := make([]byte, kdf.keyLen)
		copy(padded, key)
		return padded, nil
	}

	return key, nil
}
