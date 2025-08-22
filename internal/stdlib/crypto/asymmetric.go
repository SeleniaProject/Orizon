package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// AsymmetricAlgorithm represents supported asymmetric algorithms.
type AsymmetricAlgorithm int

const (
	RSA2048 AsymmetricAlgorithm = iota
	RSA3072
	RSA4096
	ECDSA256
	ECDSA384
	ECDSA521
	Ed25519
)

// KeyPair represents an asymmetric key pair.
type KeyPair struct {
	Algorithm  AsymmetricAlgorithm
	PrivateKey interface{}
	PublicKey  interface{}
}

// GenerateKeyPair generates a new asymmetric key pair.
func GenerateKeyPair(algo AsymmetricAlgorithm) (*KeyPair, error) {
	switch algo {
	case RSA2048:
		return generateRSAKeyPair(2048)
	case RSA3072:
		return generateRSAKeyPair(3072)
	case RSA4096:
		return generateRSAKeyPair(4096)
	case ECDSA256, ECDSA384, ECDSA521:
		return nil, errors.New("ECDSA not yet implemented")
	case Ed25519:
		return nil, errors.New("Ed25519 not yet implemented")
	default:
		return nil, errors.New("unsupported asymmetric algorithm")
	}
}

func generateRSAKeyPair(bits int) (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	var algo AsymmetricAlgorithm
	switch bits {
	case 2048:
		algo = RSA2048
	case 3072:
		algo = RSA3072
	case 4096:
		algo = RSA4096
	}

	return &KeyPair{
		Algorithm:  algo,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// Sign signs data using the private key.
func (kp *KeyPair) Sign(data []byte) ([]byte, error) {
	switch kp.Algorithm {
	case RSA2048, RSA3072, RSA4096:
		return kp.signRSA(data)
	default:
		return nil, errors.New("signing not supported for this algorithm")
	}
}

func (kp *KeyPair) signRSA(data []byte) ([]byte, error) {
	privateKey, ok := kp.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("invalid RSA private key")
	}

	hashed := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// Verify verifies a signature using the public key.
func (kp *KeyPair) Verify(data, signature []byte) error {
	switch kp.Algorithm {
	case RSA2048, RSA3072, RSA4096:
		return kp.verifyRSA(data, signature)
	default:
		return errors.New("verification not supported for this algorithm")
	}
}

func (kp *KeyPair) verifyRSA(data, signature []byte) error {
	publicKey, ok := kp.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("invalid RSA public key")
	}

	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
}

// Encrypt encrypts data using the public key.
func (kp *KeyPair) Encrypt(plaintext []byte) ([]byte, error) {
	switch kp.Algorithm {
	case RSA2048, RSA3072, RSA4096:
		return kp.encryptRSA(plaintext)
	default:
		return nil, errors.New("encryption not supported for this algorithm")
	}
}

func (kp *KeyPair) encryptRSA(plaintext []byte) ([]byte, error) {
	publicKey, ok := kp.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid RSA public key")
	}

	return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, plaintext, nil)
}

// Decrypt decrypts data using the private key.
func (kp *KeyPair) Decrypt(ciphertext []byte) ([]byte, error) {
	switch kp.Algorithm {
	case RSA2048, RSA3072, RSA4096:
		return kp.decryptRSA(ciphertext)
	default:
		return nil, errors.New("decryption not supported for this algorithm")
	}
}

func (kp *KeyPair) decryptRSA(ciphertext []byte) ([]byte, error) {
	privateKey, ok := kp.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("invalid RSA private key")
	}

	return rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
}

// ExportPrivateKeyPEM exports the private key in PEM format.
func (kp *KeyPair) ExportPrivateKeyPEM() ([]byte, error) {
	switch kp.Algorithm {
	case RSA2048, RSA3072, RSA4096:
		return kp.exportRSAPrivateKeyPEM()
	default:
		return nil, errors.New("PEM export not supported for this algorithm")
	}
}

func (kp *KeyPair) exportRSAPrivateKeyPEM() ([]byte, error) {
	privateKey, ok := kp.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("invalid RSA private key")
	}

	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return privKeyPEM, nil
}

// ExportPublicKeyPEM exports the public key in PEM format.
func (kp *KeyPair) ExportPublicKeyPEM() ([]byte, error) {
	switch kp.Algorithm {
	case RSA2048, RSA3072, RSA4096:
		return kp.exportRSAPublicKeyPEM()
	default:
		return nil, errors.New("PEM export not supported for this algorithm")
	}
}

func (kp *KeyPair) exportRSAPublicKeyPEM() ([]byte, error) {
	publicKey, ok := kp.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid RSA public key")
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return pubKeyPEM, nil
}

// ImportPrivateKeyPEM imports a private key from PEM format.
func ImportPrivateKeyPEM(pemData []byte) (*KeyPair, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		// Determine algorithm based on key size
		var algo AsymmetricAlgorithm
		keySize := privateKey.N.BitLen()
		switch keySize {
		case 2048:
			algo = RSA2048
		case 3072:
			algo = RSA3072
		case 4096:
			algo = RSA4096
		default:
			return nil, errors.New("unsupported RSA key size")
		}

		return &KeyPair{
			Algorithm:  algo,
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
		}, nil
	default:
		return nil, errors.New("unsupported private key type")
	}
}

// ImportPublicKeyPEM imports a public key from PEM format.
func ImportPublicKeyPEM(pemData []byte) (interface{}, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if block.Type == "PUBLIC KEY" {
		return x509.ParsePKIXPublicKey(block.Bytes)
	}

	return nil, errors.New("unsupported public key type")
}

// DigitalSignature provides a high-level interface for digital signatures.
type DigitalSignature struct {
	keyPair *KeyPair
}

// NewDigitalSignature creates a new digital signature instance.
func NewDigitalSignature(keyPair *KeyPair) *DigitalSignature {
	return &DigitalSignature{
		keyPair: keyPair,
	}
}

// SignData signs data and returns the signature.
func (ds *DigitalSignature) SignData(data []byte) ([]byte, error) {
	return ds.keyPair.Sign(data)
}

// VerifyData verifies a signature against data.
func (ds *DigitalSignature) VerifyData(data, signature []byte) error {
	return ds.keyPair.Verify(data, signature)
}
