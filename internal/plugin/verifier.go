package plugin

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Verifier verifies plugin signatures
type Verifier struct {
	trustedKeys []TrustedKey
	required   bool
}

// TrustedKey represents a trusted signing key
type TrustedKey struct {
	Name      string
	Algorithm string // "ed25519" or "rsa-sha256"
	PublicKey []byte
}

// NewVerifier creates a new signature verifier
func NewVerifier(keys []TrustedKey, required bool) *Verifier {
	return &Verifier{
		trustedKeys: keys,
		required:   required,
	}
}

// AddTrustedKey adds a trusted key
func (v *Verifier) AddTrustedKey(key TrustedKey) {
	v.trustedKeys = append(v.trustedKeys, key)
}

// Verify verifies a plugin's signature
// meta contains the signature and algorithm, pluginBinary is the raw .so file content
func (v *Verifier) Verify(meta *PluginMetadata, pluginBinary []byte) error {
	// If signature is empty and not required, skip verification
	if meta.Signature == "" {
		if v.required {
			return fmt.Errorf("plugin %s has no signature but signature is required", meta.Name)
		}
		return nil
	}

	if len(v.trustedKeys) == 0 {
		return fmt.Errorf("no trusted keys configured, cannot verify plugin %s", meta.Name)
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(meta.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Build content that was signed: plugin.json (excluding signature field) + binary
	// Sign content = JSON of meta WITHOUT signature field concatenated with binary
	metaCopy := *meta
	metaCopy.Signature = ""
	metaCopy.SignAlgorithm = ""
	metaJSON, _ := json.Marshal(metaCopy)

	signedContent := append(metaJSON, pluginBinary...)

	// Try each trusted key
	var lastErr error
	for _, key := range v.trustedKeys {
		var err error
		switch key.Algorithm {
		case "ed25519":
			err = v.verifyEd25519(key.PublicKey, signedContent, sigBytes)
		case "rsa-sha256":
			err = v.verifyRSA(key.PublicKey, signedContent, sigBytes)
		default:
			err = fmt.Errorf("unsupported algorithm: %s", key.Algorithm)
		}
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("signature verification failed for plugin %s: %w", meta.Name, lastErr)
}

// verifyEd25519 verifies an Ed25519 signature
func (v *Verifier) verifyEd25519(publicKeyBytes, message, signature []byte) error {
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid ed25519 public key size: got %d, want %d", len(publicKeyBytes), ed25519.PublicKeySize)
	}
	if !ed25519.Verify(publicKeyBytes, message, signature) {
		return fmt.Errorf("ed25519 signature mismatch")
	}
	return nil
}

// verifyRSA verifies an RSA-SHA256 signature
func (v *Verifier) verifyRSA(publicKeyBytes, message, signature []byte) error {
	pubKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		// Try parsing as PKCS1
		rsaKey, err2 := x509.ParsePKCS1PublicKey(publicKeyBytes)
		if err2 != nil {
			return fmt.Errorf("failed to parse RSA public key: %w (pkix: %v, pkcs1: %v)", err, err2)
		}
		pubKey = rsaKey
	}

	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	hash := sha256.Sum256(message)
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("rsa signature mismatch: %w", err)
	}
	return nil
}

// GenerateSignature generates a signature for a plugin (utility for plugin developers)
// This would be run locally by plugin developers, not by the loader
func GenerateSignature(meta *PluginMetadata, pluginBinary []byte, privateKeyBytes []byte, algorithm string) (string, error) {
	metaCopy := *meta
	metaCopy.Signature = ""
	metaCopy.SignAlgorithm = ""
	metaJSON, _ := json.Marshal(metaCopy)

	signedContent := append(metaJSON, pluginBinary...)

	var sig []byte
	var err error

	switch algorithm {
	case "ed25519":
		if len(privateKeyBytes) != ed25519.PrivateKeySize {
			return "", fmt.Errorf("invalid ed25519 private key size")
		}
		sig = ed25519.Sign(privateKeyBytes, signedContent)
	case "rsa-sha256":
		rsaKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse RSA private key: %w", err)
		}
		hash := sha256.Sum256(signedContent)
		sig, err = rsa.SignPKCS1v15(nil, rsaKey, crypto.SHA256, hash[:])
		if err != nil {
			return "", fmt.Errorf("failed to sign: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// LoadVerifierFromDir loads trusted keys from a directory
// Keys should be stored as files: <name>.<algorithm>.key (e.g., org-primary.ed25519.key)
func LoadVerifierFromDir(dir string, required bool) (*Verifier, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys directory: %w", err)
	}

	var keys []TrustedKey
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var algorithm string
		switch {
		case hasSuffix(name, ".ed25519.key"):
			algorithm = "ed25519"
			name = name[:len(name)-len(".ed25519.key")]
		case hasSuffix(name, ".rsa-sha256.key"):
			algorithm = "rsa-sha256"
			name = name[:len(name)-len(".rsa-sha256.key")]
		default:
			continue
		}

		keyBytes, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		keys = append(keys, TrustedKey{
			Name:      name,
			Algorithm: algorithm,
			PublicKey: keyBytes,
		})
	}

	return NewVerifier(keys, required), nil
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
