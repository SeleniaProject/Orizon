package crypto

import (
	"encoding/hex"
	"testing"
)

func TestSHA256Hash(t *testing.T) {
	data := []byte("Hello, Orizon!")
	expected := "8c7dd922ad47494fc02c388e12c00eac67cb6f0d0c2dc8e9d0f3c3b8d8d9e0a8"

	result := SHA256SumHex(data)
	if result != expected {
		t.Errorf("SHA256SumHex() = %s, want %s", result, expected)
	}
}

func TestHashInterface(t *testing.T) {
	h, err := NewHash(SHA256)
	if err != nil {
		t.Fatalf("NewHash() error = %v", err)
	}

	data := []byte("test data")
	h.Write(data)

	sum1 := h.SumHex()
	h.Reset()
	h.Write(data)
	sum2 := h.SumHex()

	if sum1 != sum2 {
		t.Errorf("Hash results should be consistent: %s != %s", sum1, sum2)
	}
}

func TestHMAC(t *testing.T) {
	key := []byte("secret-key")
	data := []byte("message to authenticate")

	hmac := NewHMAC(key, SHA256)

	signature, err := hmac.Sign(data)
	if err != nil {
		t.Fatalf("HMAC.Sign() error = %v", err)
	}

	valid, err := hmac.Verify(data, signature)
	if err != nil {
		t.Fatalf("HMAC.Verify() error = %v", err)
	}

	if !valid {
		t.Error("HMAC verification failed for valid signature")
	}

	// Test with wrong data
	wrongData := []byte("wrong message")
	valid, err = hmac.Verify(wrongData, signature)
	if err != nil {
		t.Fatalf("HMAC.Verify() error = %v", err)
	}

	if valid {
		t.Error("HMAC verification should fail for wrong data")
	}
}

func TestAESEncryption(t *testing.T) {
	key, err := GenerateAESKey(256)
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}

	plaintext := []byte("This is a secret message for AES encryption test")

	cipher, err := NewCipher(AES256, key)
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}

	ciphertext, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := cipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match original: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestRSAKeyGeneration(t *testing.T) {
	keyPair, err := GenerateKeyPair(RSA2048)
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if keyPair.Algorithm != RSA2048 {
		t.Errorf("Algorithm = %v, want %v", keyPair.Algorithm, RSA2048)
	}

	if keyPair.PrivateKey == nil || keyPair.PublicKey == nil {
		t.Error("Generated key pair has nil keys")
	}
}

func TestRSASignature(t *testing.T) {
	keyPair, err := GenerateKeyPair(RSA2048)
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	data := []byte("Document to be signed")

	signature, err := keyPair.Sign(data)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	err = keyPair.Verify(data, signature)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}

	// Test with wrong data
	wrongData := []byte("Wrong document")
	err = keyPair.Verify(wrongData, signature)
	if err == nil {
		t.Error("Verify() should fail for wrong data")
	}
}

func TestRSAEncryption(t *testing.T) {
	keyPair, err := GenerateKeyPair(RSA2048)
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	plaintext := []byte("Secret message")

	ciphertext, err := keyPair.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := keyPair.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match original: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestPEMExportImport(t *testing.T) {
	keyPair, err := GenerateKeyPair(RSA2048)
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Export private key
	privPEM, err := keyPair.ExportPrivateKeyPEM()
	if err != nil {
		t.Fatalf("ExportPrivateKeyPEM() error = %v", err)
	}

	// Export public key
	pubPEM, err := keyPair.ExportPublicKeyPEM()
	if err != nil {
		t.Fatalf("ExportPublicKeyPEM() error = %v", err)
	}

	// Import private key
	importedKeyPair, err := ImportPrivateKeyPEM(privPEM)
	if err != nil {
		t.Fatalf("ImportPrivateKeyPEM() error = %v", err)
	}

	// Test that imported key works
	data := []byte("Test message")
	signature, err := importedKeyPair.Sign(data)
	if err != nil {
		t.Fatalf("Sign() with imported key error = %v", err)
	}

	err = importedKeyPair.Verify(data, signature)
	if err != nil {
		t.Errorf("Verify() with imported key error = %v", err)
	}

	// Import public key
	_, err = ImportPublicKeyPEM(pubPEM)
	if err != nil {
		t.Fatalf("ImportPublicKeyPEM() error = %v", err)
	}
}

func TestKDF(t *testing.T) {
	password := []byte("user-password")
	salt := []byte("random-salt-1234567890")

	kdf := NewKDF(1000, 32)

	key1, err := kdf.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	key2, err := kdf.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("Derived key length = %d, want 32", len(key1))
	}

	if hex.EncodeToString(key1) != hex.EncodeToString(key2) {
		t.Error("KDF should produce consistent results")
	}

	// Test with different salt
	differentSalt := []byte("different-salt-1234567890")
	key3, err := kdf.DeriveKey(password, differentSalt)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	if hex.EncodeToString(key1) == hex.EncodeToString(key3) {
		t.Error("Different salts should produce different keys")
	}
}

func TestRandomGeneration(t *testing.T) {
	rng := NewRandom()

	// Test byte generation
	bytes1, err := rng.Bytes(16)
	if err != nil {
		t.Fatalf("Random.Bytes() error = %v", err)
	}

	bytes2, err := rng.Bytes(16)
	if err != nil {
		t.Fatalf("Random.Bytes() error = %v", err)
	}

	if len(bytes1) != 16 || len(bytes2) != 16 {
		t.Error("Random bytes should have correct length")
	}

	if hex.EncodeToString(bytes1) == hex.EncodeToString(bytes2) {
		t.Error("Random bytes should be different")
	}
}

func BenchmarkSHA256(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SHA256Sum(data)
	}
}

func BenchmarkAESEncrypt(b *testing.B) {
	key, _ := GenerateAESKey(256)
	cipher, _ := NewCipher(AES256, key)
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cipher.Encrypt(data)
	}
}
