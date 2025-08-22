// Package crypto provides comprehensive cryptographic functions and security utilities.
// This package includes symmetric and asymmetric encryption, digital signatures,
// hash functions, key derivation, random number generation, and advanced cryptographic protocols.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"
	"time"
)

// Hash represents different hash algorithms.
type Hash int

const (
	SHA256 Hash = iota
	SHA512
	SHA3_256
	SHA3_512
	BLAKE2b
	BLAKE2s
	MD5
	SHA1
)

// SymmetricAlgorithm represents symmetric encryption algorithms.
type SymmetricAlgorithm int

const (
	AES128 SymmetricAlgorithm = iota
	AES192
	AES256
	ChaCha20
	Salsa20
	Blowfish
	Twofish
	DES
	TripleDES
)

// AsymmetricAlgorithm represents asymmetric encryption algorithms.
type AsymmetricAlgorithm int

const (
	RSA2048 AsymmetricAlgorithm = iota
	RSA3072
	RSA4096
	ECDSA_P256
	ECDSA_P384
	ECDSA_P521
	Ed25519
	X25519
)

// KeyDerivationFunction represents key derivation functions.
type KeyDerivationFunction int

const (
	PBKDF2 KeyDerivationFunction = iota
	Scrypt
	Argon2
	Bcrypt
	HKDF
)

// CipherMode represents encryption modes.
type CipherMode int

const (
	CBC CipherMode = iota
	CFB
	OFB
	CTR
	GCM
	CCM
	EAX
	OCB
)

// PrivateKey represents a private key.
type PrivateKey struct {
	Algorithm AsymmetricAlgorithm
	KeySize   int
	Key       interface{}
	CreatedAt time.Time
}

// PublicKey represents a public key.
type PublicKey struct {
	Algorithm AsymmetricAlgorithm
	KeySize   int
	Key       interface{}
	CreatedAt time.Time
}

// KeyPair represents a public-private key pair.
type KeyPair struct {
	Private *PrivateKey
	Public  *PublicKey
}

// SymmetricKey represents a symmetric encryption key.
type SymmetricKey struct {
	Algorithm SymmetricAlgorithm
	Key       []byte
	IV        []byte
	CreatedAt time.Time
}

// Certificate represents a digital certificate.
type Certificate struct {
	Subject      CertificateSubject
	Issuer       CertificateSubject
	SerialNumber *big.Int
	NotBefore    time.Time
	NotAfter     time.Time
	KeyUsage     KeyUsage
	Extensions   []Extension
	PublicKey    *PublicKey
	Signature    []byte
	Raw          []byte
}

// CertificateSubject represents certificate subject information.
type CertificateSubject struct {
	CommonName         string
	Organization       []string
	OrganizationalUnit []string
	Country            []string
	Province           []string
	Locality           []string
	StreetAddress      []string
	PostalCode         []string
}

// KeyUsage represents certificate key usage.
type KeyUsage int

const (
	DigitalSignature KeyUsage = 1 << iota
	KeyEncipherment
	DataEncipherment
	KeyAgreement
	CertSign
	CRLSign
	EncipherOnly
	DecipherOnly
)

// Extension represents a certificate extension.
type Extension struct {
	ID       []int
	Critical bool
	Value    []byte
}

// Hasher provides hashing functionality.
type Hasher struct {
	Algorithm Hash
	hasher    hash.Hash
}

// HMAC provides HMAC functionality.
type HMAC struct {
	Algorithm Hash
	Key       []byte
	hasher    hash.Hash
}

// Signer provides digital signature functionality.
type Signer struct {
	PrivateKey *PrivateKey
	Hash       Hash
}

// Verifier provides signature verification functionality.
type Verifier struct {
	PublicKey *PublicKey
	Hash      Hash
}

// Encryptor provides encryption functionality.
type Encryptor struct {
	Key       interface{}
	Algorithm interface{}
	Mode      CipherMode
}

// Decryptor provides decryption functionality.
type Decryptor struct {
	Key       interface{}
	Algorithm interface{}
	Mode      CipherMode
}

// RandomGenerator provides secure random number generation.
type RandomGenerator struct {
	Source io.Reader
}

// SecureRandom provides cryptographically secure random operations.
type SecureRandom struct {
	generator *RandomGenerator
}

// KDF provides key derivation functionality.
type KDF struct {
	Function   KeyDerivationFunction
	Salt       []byte
	Iterations int
	KeyLength  int
	Memory     int // For Argon2
	Threads    int // For Argon2
}

// TLS represents TLS configuration and operations.
type TLS struct {
	Version      TLSVersion
	CipherSuites []CipherSuite
	Certificates []*Certificate
	PrivateKeys  []*PrivateKey
	RootCAs      []*Certificate
	ClientAuth   ClientAuthType
	MinVersion   TLSVersion
	MaxVersion   TLSVersion
}

// TLSVersion represents TLS versions.
type TLSVersion int

const (
	TLS10 TLSVersion = iota
	TLS11
	TLS12
	TLS13
)

// CipherSuite represents TLS cipher suites.
type CipherSuite int

const (
	TLS_AES_128_GCM_SHA256 CipherSuite = iota
	TLS_AES_256_GCM_SHA384
	TLS_CHACHA20_POLY1305_SHA256
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
)

// ClientAuthType represents client authentication types.
type ClientAuthType int

const (
	NoClientCert ClientAuthType = iota
	RequestClientCert
	RequireAnyClientCert
	VerifyClientCertIfGiven
	RequireAndVerifyClientCert
)

// ZKProof represents zero-knowledge proof systems.
type ZKProof struct {
	Type         ZKProofType
	Proof        []byte
	PublicInputs [][]byte
	Circuit      *Circuit
}

// ZKProofType represents types of zero-knowledge proofs.
type ZKProofType int

const (
	Groth16 ZKProofType = iota
	PLONK
	STARK
	Bulletproofs
	zkSNARKs
	zkSTARKs
)

// Circuit represents a zero-knowledge proof circuit.
type Circuit struct {
	Gates       []Gate
	Constraints []Constraint
	Inputs      []Input
	Outputs     []Output
}

// Gate represents a circuit gate.
type Gate struct {
	Type   GateType
	Inputs []int
	Output int
	Value  *big.Int
}

// GateType represents types of circuit gates.
type GateType int

const (
	AndGate GateType = iota
	OrGate
	XorGate
	NotGate
	AddGate
	MulGate
)

// Constraint represents a circuit constraint.
type Constraint struct {
	A [][]int
	B [][]int
	C [][]int
}

// Input represents a circuit input.
type Input struct {
	Name  string
	Type  string
	Value *big.Int
}

// Output represents a circuit output.
type Output struct {
	Name  string
	Type  string
	Value *big.Int
}

// MultipartyComputation provides secure multi-party computation.
type MultipartyComputation struct {
	Parties  []Party
	Protocol MPCProtocol
	Function *Function
	Shares   map[string]*SecretShare
	Results  map[string]*big.Int
}

// Party represents a party in MPC.
type Party struct {
	ID        string
	PublicKey *PublicKey
	Address   string
	Online    bool
}

// MPCProtocol represents MPC protocols.
type MPCProtocol int

const (
	ShamirSecretSharing MPCProtocol = iota
	BGW
	GMW
	ABY
)

// Function represents a function for MPC.
type Function struct {
	Name        string
	Inputs      []string
	Outputs     []string
	Circuit     *Circuit
	Description string
}

// SecretShare represents a secret share.
type SecretShare struct {
	PartyID   string
	Value     *big.Int
	Threshold int
	Total     int
}

// HomomorphicEncryption provides homomorphic encryption capabilities.
type HomomorphicEncryption struct {
	Scheme     HEScheme
	PublicKey  *PublicKey
	PrivateKey *PrivateKey
	Parameters HEParameters
}

// HEScheme represents homomorphic encryption schemes.
type HEScheme int

const (
	BFV HEScheme = iota
	BGV
	CKKS
	TFHE
	FHEW
)

// HEParameters represents homomorphic encryption parameters.
type HEParameters struct {
	PolyDegree    int
	PlainModulus  *big.Int
	CipherModulus []*big.Int
	Sigma         float64
}

// Blockchain provides blockchain cryptography utilities.
type Blockchain struct {
	HashAlgorithm   Hash
	SignatureScheme AsymmetricAlgorithm
	MerkleTree      *MerkleTree
	Consensus       ConsensusAlgorithm
}

// MerkleTree represents a Merkle tree.
type MerkleTree struct {
	Root   *MerkleNode
	Leaves []*MerkleNode
	Hash   Hash
	Height int
}

// MerkleNode represents a node in a Merkle tree.
type MerkleNode struct {
	Hash   []byte
	Left   *MerkleNode
	Right  *MerkleNode
	Parent *MerkleNode
	Data   []byte
}

// ConsensusAlgorithm represents blockchain consensus algorithms.
type ConsensusAlgorithm int

const (
	ProofOfWork ConsensusAlgorithm = iota
	ProofOfStake
	DelegatedProofOfStake
	PracticalByzantineFaultTolerance
	HoneyBadgerBFT
)

// QuantumResistant provides post-quantum cryptography.
type QuantumResistant struct {
	KeyExchange PQKeyExchange
	Signature   PQSignature
	Encryption  PQEncryption
	Parameters  PQParameters
}

// PQKeyExchange represents post-quantum key exchange algorithms.
type PQKeyExchange int

const (
	Kyber PQKeyExchange = iota
	NTRU
	NewHope
	FrodoKEM
	SIKE
)

// PQSignature represents post-quantum signature algorithms.
type PQSignature int

const (
	Dilithium PQSignature = iota
	Falcon
	Rainbow
	SPHINCS
)

// PQEncryption represents post-quantum encryption algorithms.
type PQEncryption int

const (
	McEliece PQEncryption = iota
	BIKE
	HQC
	LEDAcrypt
)

// PQParameters represents post-quantum cryptography parameters.
type PQParameters struct {
	SecurityLevel  int
	KeySize        int
	SignatureSize  int
	PublicKeySize  int
	PrivateKeySize int
}

// Hash Functions Implementation

// NewHasher creates a new hasher for the specified algorithm.
func NewHasher(algorithm Hash) (*Hasher, error) {
	hasher := &Hasher{Algorithm: algorithm}

	switch algorithm {
	case SHA256:
		hasher.hasher = sha256.New()
	case SHA512:
		hasher.hasher = sha512.New()
	default:
		return nil, errors.New("unsupported hash algorithm")
	}

	return hasher, nil
}

// Hash computes the hash of data.
func (h *Hasher) Hash(data []byte) []byte {
	h.hasher.Reset()
	h.hasher.Write(data)
	return h.hasher.Sum(nil)
}

// HashString computes the hash of a string.
func (h *Hasher) HashString(data string) string {
	hash := h.Hash([]byte(data))
	return hex.EncodeToString(hash)
}

// NewHMAC creates a new HMAC instance.
func NewHMAC(algorithm Hash, key []byte) (*HMAC, error) {
	hmac := &HMAC{Algorithm: algorithm, Key: key}

	switch algorithm {
	case SHA256:
		hmac.hasher = sha256.New()
	case SHA512:
		hmac.hasher = sha512.New()
	default:
		return nil, errors.New("unsupported hash algorithm")
	}

	return hmac, nil
}

// Compute computes the HMAC of data.
func (h *HMAC) Compute(data []byte) []byte {
	// Simplified HMAC implementation
	if len(h.Key) > 64 {
		hasher := sha256.New()
		hasher.Write(h.Key)
		h.Key = hasher.Sum(nil)
	}

	// Pad key to block size
	key := make([]byte, 64)
	copy(key, h.Key)

	// Create inner and outer padding
	ipad := make([]byte, 64)
	opad := make([]byte, 64)

	for i := 0; i < 64; i++ {
		ipad[i] = key[i] ^ 0x36
		opad[i] = key[i] ^ 0x5c
	}

	// Inner hash
	h.hasher.Reset()
	h.hasher.Write(ipad)
	h.hasher.Write(data)
	innerHash := h.hasher.Sum(nil)

	// Outer hash
	h.hasher.Reset()
	h.hasher.Write(opad)
	h.hasher.Write(innerHash)

	return h.hasher.Sum(nil)
}

// Key Generation Implementation

// GenerateKeyPair generates a new key pair.
func GenerateKeyPair(algorithm AsymmetricAlgorithm) (*KeyPair, error) {
	switch algorithm {
	case RSA2048:
		return generateRSAKeyPair(2048)
	case RSA3072:
		return generateRSAKeyPair(3072)
	case RSA4096:
		return generateRSAKeyPair(4096)
	default:
		return nil, errors.New("unsupported asymmetric algorithm")
	}
}

// generateRSAKeyPair generates an RSA key pair.
func generateRSAKeyPair(bits int) (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	priv := &PrivateKey{
		Algorithm: RSA2048,
		KeySize:   bits,
		Key:       privateKey,
		CreatedAt: now,
	}

	pub := &PublicKey{
		Algorithm: RSA2048,
		KeySize:   bits,
		Key:       &privateKey.PublicKey,
		CreatedAt: now,
	}

	return &KeyPair{Private: priv, Public: pub}, nil
}

// GenerateSymmetricKey generates a symmetric encryption key.
func GenerateSymmetricKey(algorithm SymmetricAlgorithm) (*SymmetricKey, error) {
	var keySize int

	switch algorithm {
	case AES128:
		keySize = 16
	case AES192:
		keySize = 24
	case AES256:
		keySize = 32
	default:
		return nil, errors.New("unsupported symmetric algorithm")
	}

	key := make([]byte, keySize)
	iv := make([]byte, aes.BlockSize)

	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	return &SymmetricKey{
		Algorithm: algorithm,
		Key:       key,
		IV:        iv,
		CreatedAt: time.Now(),
	}, nil
}

// Encryption Implementation

// NewEncryptor creates a new encryptor.
func NewEncryptor(key interface{}, algorithm interface{}, mode CipherMode) *Encryptor {
	return &Encryptor{
		Key:       key,
		Algorithm: algorithm,
		Mode:      mode,
	}
}

// Encrypt encrypts data.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	switch key := e.Key.(type) {
	case *SymmetricKey:
		return e.encryptSymmetric(plaintext, key)
	case *PublicKey:
		return e.encryptAsymmetric(plaintext, key)
	default:
		return nil, errors.New("unsupported key type")
	}
}

// encryptSymmetric performs symmetric encryption.
func (e *Encryptor) encryptSymmetric(plaintext []byte, key *SymmetricKey) ([]byte, error) {
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, err
	}

	// Pad plaintext to block size
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padtext := make([]byte, len(plaintext)+padding)
	copy(padtext, plaintext)
	for i := len(plaintext); i < len(padtext); i++ {
		padtext[i] = byte(padding)
	}

	ciphertext := make([]byte, len(padtext))

	switch e.Mode {
	case CBC:
		mode := cipher.NewCBCEncryptor(block, key.IV)
		mode.CryptBlocks(ciphertext, padtext)
	case CFB:
		mode := cipher.NewCFBEncrypter(block, key.IV)
		mode.XORKeyStream(ciphertext, padtext)
	case OFB:
		mode := cipher.NewOFB(block, key.IV)
		mode.XORKeyStream(ciphertext, padtext)
	case CTR:
		mode := cipher.NewCTR(block, key.IV)
		mode.XORKeyStream(ciphertext, padtext)
	default:
		return nil, errors.New("unsupported cipher mode")
	}

	return ciphertext, nil
}

// encryptAsymmetric performs asymmetric encryption.
func (e *Encryptor) encryptAsymmetric(plaintext []byte, key *PublicKey) ([]byte, error) {
	switch pubKey := key.Key.(type) {
	case *rsa.PublicKey:
		return rsa.EncryptPKCS1v15(rand.Reader, pubKey, plaintext)
	default:
		return nil, errors.New("unsupported public key type")
	}
}

// NewDecryptor creates a new decryptor.
func NewDecryptor(key interface{}, algorithm interface{}, mode CipherMode) *Decryptor {
	return &Decryptor{
		Key:       key,
		Algorithm: algorithm,
		Mode:      mode,
	}
}

// Decrypt decrypts data.
func (d *Decryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	switch key := d.Key.(type) {
	case *SymmetricKey:
		return d.decryptSymmetric(ciphertext, key)
	case *PrivateKey:
		return d.decryptAsymmetric(ciphertext, key)
	default:
		return nil, errors.New("unsupported key type")
	}
}

// decryptSymmetric performs symmetric decryption.
func (d *Decryptor) decryptSymmetric(ciphertext []byte, key *SymmetricKey) ([]byte, error) {
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))

	switch d.Mode {
	case CBC:
		mode := cipher.NewCBCDecryptor(block, key.IV)
		mode.CryptBlocks(plaintext, ciphertext)
	case CFB:
		mode := cipher.NewCFBDecrypter(block, key.IV)
		mode.XORKeyStream(plaintext, ciphertext)
	case OFB:
		mode := cipher.NewOFB(block, key.IV)
		mode.XORKeyStream(plaintext, ciphertext)
	case CTR:
		mode := cipher.NewCTR(block, key.IV)
		mode.XORKeyStream(plaintext, ciphertext)
	default:
		return nil, errors.New("unsupported cipher mode")
	}

	// Remove padding
	if d.Mode == CBC {
		padding := int(plaintext[len(plaintext)-1])
		plaintext = plaintext[:len(plaintext)-padding]
	}

	return plaintext, nil
}

// decryptAsymmetric performs asymmetric decryption.
func (d *Decryptor) decryptAsymmetric(ciphertext []byte, key *PrivateKey) ([]byte, error) {
	switch privKey := key.Key.(type) {
	case *rsa.PrivateKey:
		return rsa.DecryptPKCS1v15(rand.Reader, privKey, ciphertext)
	default:
		return nil, errors.New("unsupported private key type")
	}
}

// Digital Signatures Implementation

// NewSigner creates a new signer.
func NewSigner(privateKey *PrivateKey, hashAlgorithm Hash) *Signer {
	return &Signer{
		PrivateKey: privateKey,
		Hash:       hashAlgorithm,
	}
}

// Sign signs data.
func (s *Signer) Sign(data []byte) ([]byte, error) {
	hasher, err := NewHasher(s.Hash)
	if err != nil {
		return nil, err
	}

	hash := hasher.Hash(data)

	switch privKey := s.PrivateKey.Key.(type) {
	case *rsa.PrivateKey:
		return rsa.SignPKCS1v15(rand.Reader, privKey, 0, hash)
	default:
		return nil, errors.New("unsupported private key type")
	}
}

// NewVerifier creates a new verifier.
func NewVerifier(publicKey *PublicKey, hashAlgorithm Hash) *Verifier {
	return &Verifier{
		PublicKey: publicKey,
		Hash:      hashAlgorithm,
	}
}

// Verify verifies a signature.
func (v *Verifier) Verify(data, signature []byte) error {
	hasher, err := NewHasher(v.Hash)
	if err != nil {
		return err
	}

	hash := hasher.Hash(data)

	switch pubKey := v.PublicKey.Key.(type) {
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(pubKey, 0, hash, signature)
	default:
		return errors.New("unsupported public key type")
	}
}

// Random Number Generation

// NewSecureRandom creates a new secure random generator.
func NewSecureRandom() *SecureRandom {
	return &SecureRandom{
		generator: &RandomGenerator{Source: rand.Reader},
	}
}

// Bytes generates random bytes.
func (sr *SecureRandom) Bytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := sr.generator.Source.Read(bytes)
	return bytes, err
}

// Int generates a random integer in range [0, max).
func (sr *SecureRandom) Int(max int64) (int64, error) {
	bigMax := big.NewInt(max)
	result, err := rand.Int(rand.Reader, bigMax)
	if err != nil {
		return 0, err
	}
	return result.Int64(), nil
}

// String generates a random string.
func (sr *SecureRandom) String(length int, charset string) (string, error) {
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	bytes := make([]byte, length)
	for i := range bytes {
		idx, err := sr.Int(int64(len(charset)))
		if err != nil {
			return "", err
		}
		bytes[i] = charset[idx]
	}

	return string(bytes), nil
}

// Key Derivation Functions

// NewKDF creates a new key derivation function.
func NewKDF(function KeyDerivationFunction, salt []byte, iterations, keyLength int) *KDF {
	return &KDF{
		Function:   function,
		Salt:       salt,
		Iterations: iterations,
		KeyLength:  keyLength,
	}
}

// DeriveKey derives a key from a password.
func (kdf *KDF) DeriveKey(password []byte) ([]byte, error) {
	switch kdf.Function {
	case PBKDF2:
		return kdf.pbkdf2(password), nil
	default:
		return nil, errors.New("unsupported KDF function")
	}
}

// pbkdf2 implements PBKDF2 key derivation.
func (kdf *KDF) pbkdf2(password []byte) []byte {
	// Simplified PBKDF2 implementation
	dk := make([]byte, kdf.KeyLength)

	for i := 0; i < kdf.KeyLength; i += 32 {
		hmac, _ := NewHMAC(SHA256, password)

		// Create U1
		salt := append(kdf.Salt, []byte{byte(i/32 + 1)}...)
		u := hmac.Compute(salt)
		result := make([]byte, len(u))
		copy(result, u)

		// Iterate
		for j := 1; j < kdf.Iterations; j++ {
			u = hmac.Compute(u)
			for k := range result {
				result[k] ^= u[k]
			}
		}

		// Copy to output
		copy(dk[i:], result)
	}

	return dk[:kdf.KeyLength]
}

// Utility Functions

// EncodeBase64 encodes bytes to base64.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 to bytes.
func DecodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// EncodeHex encodes bytes to hexadecimal.
func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

// DecodeHex decodes hexadecimal to bytes.
func DecodeHex(data string) ([]byte, error) {
	return hex.DecodeString(data)
}

// EncodePEM encodes a key to PEM format.
func EncodePEM(key interface{}, keyType string) ([]byte, error) {
	var der []byte
	var err error

	switch k := key.(type) {
	case *rsa.PrivateKey:
		der = x509.MarshalPKCS1PrivateKey(k)
	case *rsa.PublicKey:
		der, err = x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported key type")
	}

	block := &pem.Block{
		Type:  keyType,
		Bytes: der,
	}

	return pem.EncodeToMemory(block), nil
}

// DecodePEM decodes a PEM encoded key.
func DecodePEM(data []byte) (interface{}, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PUBLIC KEY":
		return x509.ParsePKIXPublicKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported PEM type: %s", block.Type)
	}
}

// CompareHashes compares two hashes in constant time.
func CompareHashes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

// ZeroBytes securely zeros a byte slice.
func ZeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
