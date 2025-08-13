package packagemanager

import (
    "context"
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "sort"
    "time"
)

// KeyID is a stable identifier for a public key, derived from its SHA-256 hash.
type KeyID string

// Fingerprint computes the KeyID (sha256 hex) of a raw public key.
func Fingerprint(pub ed25519.PublicKey) KeyID {
    sum := sha256.Sum256(pub)
    return KeyID(hex.EncodeToString(sum[:]))
}

// Certificate represents a minimal, portable certificate for Ed25519 keys.
// It is intentionally simple and JSON-serializable without external dependencies.
type Certificate struct {
    Serial      string            `json:"serial"`
    Subject     string            `json:"subject"`
    Issuer      string            `json:"issuer"`
    PublicKey   []byte            `json:"public_key"`
    NotBefore   time.Time         `json:"not_before"`
    NotAfter    time.Time         `json:"not_after"`
    KeyUsage    []string          `json:"key_usage,omitempty"`
    Extensions  map[string]string `json:"extensions,omitempty"`
    // Signature is the issuer's detached signature over the certificate TBSCertificate.
    Signature   []byte            `json:"signature,omitempty"`
}

// tbsCertificate returns the canonical bytes of the certificate fields that are signed by issuer.
func (c *Certificate) tbsCertificate() ([]byte, error) {
    // Note: The order of fields is fixed by the struct and json encoder.
    tmp := struct {
        Serial     string            `json:"serial"`
        Subject    string            `json:"subject"`
        Issuer     string            `json:"issuer"`
        PublicKey  []byte            `json:"public_key"`
        NotBefore  time.Time         `json:"not_before"`
        NotAfter   time.Time         `json:"not_after"`
        KeyUsage   []string          `json:"key_usage,omitempty"`
        Extensions map[string]string `json:"extensions,omitempty"`
    }{
        Serial: c.Serial,
        Subject: c.Subject,
        Issuer: c.Issuer,
        PublicKey: c.PublicKey,
        NotBefore: c.NotBefore,
        NotAfter: c.NotAfter,
        KeyUsage: append([]string(nil), c.KeyUsage...),
        Extensions: copyStringMap(c.Extensions),
    }
    // Ensure determinism: sort KeyUsage, Extensions
    sort.Strings(tmp.KeyUsage)
    if tmp.Extensions != nil {
        // Marshal to ensure stable order of keys
        // We will re-marshal after sorting keys by moving into a slice
        keys := make([]string, 0, len(tmp.Extensions))
        for k := range tmp.Extensions { keys = append(keys, k) }
        sort.Strings(keys)
        ordered := make(map[string]string, len(keys))
        for _, k := range keys { ordered[k] = tmp.Extensions[k] }
        tmp.Extensions = ordered
    }
    return json.Marshal(tmp)
}

// TrustStore holds trusted roots and optional intermediates for chain validation.
type TrustStore struct {
    roots        map[KeyID]ed25519.PublicKey
    intermediates map[KeyID]Certificate
}

// NewTrustStore constructs an empty TrustStore.
func NewTrustStore() *TrustStore {
    return &TrustStore{ roots: make(map[KeyID]ed25519.PublicKey), intermediates: make(map[KeyID]Certificate) }
}

// AddRoot adds a trusted root public key.
func (ts *TrustStore) AddRoot(pub ed25519.PublicKey) KeyID {
    kid := Fingerprint(pub)
    ts.roots[kid] = append(ed25519.PublicKey(nil), pub...)
    return kid
}

// AddIntermediate registers an intermediate certificate (issuer is either a root or another intermediate).
func (ts *TrustStore) AddIntermediate(cert Certificate) {
    kid := Fingerprint(ed25519.PublicKey(cert.PublicKey))
    ts.intermediates[kid] = cert
}

// GenerateEd25519Keypair creates a new Ed25519 key pair.
func GenerateEd25519Keypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { return nil, nil, err }
    return pub, priv, nil
}

// SelfSignRoot creates a self-signed root certificate for the given key pair.
func SelfSignRoot(subject string, pub ed25519.PublicKey, priv ed25519.PrivateKey, validity time.Duration) (Certificate, error) {
    cert := Certificate{
        Serial:    randomSerial(),
        Subject:   subject,
        Issuer:    subject,
        PublicKey: append([]byte(nil), pub...),
        NotBefore: time.Now().Add(-time.Minute),
        NotAfter:  time.Now().Add(validity),
        KeyUsage:  []string{"cert-sign", "package-sign", "lockfile-sign"},
    }
    tbs, err := cert.tbsCertificate()
    if err != nil { return Certificate{}, err }
    cert.Signature = ed25519.Sign(priv, tbs)
    return cert, nil
}

// IssueChild creates and signs a child certificate for childPub using parent certificate+private key.
func IssueChild(parent Certificate, parentPriv ed25519.PrivateKey, childPub ed25519.PublicKey, subject string, validity time.Duration, usages []string) (Certificate, error) {
    cert := Certificate{
        Serial:    randomSerial(),
        Subject:   subject,
        Issuer:    parent.Subject,
        PublicKey: append([]byte(nil), childPub...),
        NotBefore: time.Now().Add(-time.Minute),
        NotAfter:  time.Now().Add(validity),
        KeyUsage:  append([]string(nil), usages...),
    }
    tbs, err := cert.tbsCertificate()
    if err != nil { return Certificate{}, err }
    cert.Signature = ed25519.Sign(parentPriv, tbs)
    return cert, nil
}

// VerifyCertificate verifies a single certificate against the issuer public key.
func VerifyCertificate(cert Certificate, issuerPub ed25519.PublicKey) error {
    tbs, err := cert.tbsCertificate()
    if err != nil { return err }
    if !ed25519.Verify(issuerPub, tbs, cert.Signature) {
        return errors.New("certificate signature invalid")
    }
    now := time.Now()
    if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
        return errors.New("certificate is not within validity period")
    }
    return nil
}

// VerifyChain validates a chain [leaf, ..., root], ensuring each is signed by the next, ending in a trusted root.
func (ts *TrustStore) VerifyChain(chain []Certificate) error {
    if len(chain) == 0 { return errors.New("empty certificate chain") }
    // Traverse up the chain
    for i := 0; i < len(chain)-1; i++ {
        issuerPub := ed25519.PublicKey(chain[i+1].PublicKey)
        if err := VerifyCertificate(chain[i], issuerPub); err != nil {
            return fmt.Errorf("chain[%d] verification failed: %w", i, err)
        }
    }
    // The last cert must be a trusted root
    root := chain[len(chain)-1]
    rootPub := ed25519.PublicKey(root.PublicKey)
    if err := VerifyCertificate(root, rootPub); err != nil {
        return fmt.Errorf("root self-signature invalid: %w", err)
    }
    if _, ok := ts.roots[Fingerprint(rootPub)]; !ok {
        return errors.New("root is not trusted")
    }
    return nil
}

// PackageDescriptor is the canonical content that is signed for package blobs.
type PackageDescriptor struct {
    Name         PackageID     `json:"name"`
    Version      Version       `json:"version"`
    CID          CID           `json:"cid"`
    SHA256       string        `json:"sha256"`
    Dependencies []Dependency  `json:"dependencies,omitempty"`
}

// descriptorBytes produces canonical JSON for signing.
func descriptorBytes(d PackageDescriptor) ([]byte, error) {
    // Ensure deterministic dependency ordering
    deps := append([]Dependency(nil), d.Dependencies...)
    sort.Slice(deps, func(i, j int) bool {
        if deps[i].Name != deps[j].Name { return deps[i].Name < deps[j].Name }
        return deps[i].Constraint < deps[j].Constraint
    })
    type canon struct {
        Name    PackageID    `json:"name"`
        Version Version      `json:"version"`
        CID     CID          `json:"cid"`
        SHA256  string       `json:"sha256"`
        Deps    []Dependency `json:"dependencies,omitempty"`
    }
    obj := canon{ Name: d.Name, Version: d.Version, CID: d.CID, SHA256: d.SHA256, Deps: deps }
    return json.Marshal(obj)
}

// BuildDescriptor fetches the blob from registry and computes its descriptor.
func BuildDescriptor(ctx context.Context, reg Registry, cid CID) (PackageDescriptor, error) {
    blob, err := reg.Fetch(ctx, cid)
    if err != nil { return PackageDescriptor{}, err }
    sum := sha256.Sum256(blob.Data)
    return PackageDescriptor{
        Name: blob.Manifest.Name,
        Version: blob.Manifest.Version,
        CID: cid,
        SHA256: hex.EncodeToString(sum[:]),
        Dependencies: append([]Dependency(nil), blob.Manifest.Dependencies...),
    }, nil
}

// SignatureBundle is a detached signature along with the certificate chain used for signing.
type SignatureBundle struct {
    Algorithm  string        `json:"algorithm"`
    KeyID      KeyID         `json:"key_id"`
    Signature  []byte        `json:"signature"`
    Chain      []Certificate `json:"chain"`
}

// SignDescriptor produces a signature bundle for a descriptor using the provided private key and chain.
func SignDescriptor(desc PackageDescriptor, signerPriv ed25519.PrivateKey, chain []Certificate) (SignatureBundle, error) {
    b, err := descriptorBytes(desc)
    if err != nil { return SignatureBundle{}, err }
    sig := ed25519.Sign(signerPriv, b)
    var leafPub ed25519.PublicKey
    if len(chain) == 0 { return SignatureBundle{}, errors.New("missing certificate chain") }
    leafPub = ed25519.PublicKey(chain[0].PublicKey)
    return SignatureBundle{
        Algorithm: "ed25519",
        KeyID: Fingerprint(leafPub),
        Signature: sig,
        Chain: append([]Certificate(nil), chain...),
    }, nil
}

// VerifyDescriptor verifies the descriptor against the signature bundle using the trust store.
func (ts *TrustStore) VerifyDescriptor(desc PackageDescriptor, bundle SignatureBundle) error {
    if bundle.Algorithm != "ed25519" { return errors.New("unsupported algorithm") }
    if len(bundle.Chain) == 0 { return errors.New("empty certificate chain") }
    // Validate chain
    if err := ts.VerifyChain(bundle.Chain); err != nil { return err }
    // Verify signature
    b, err := descriptorBytes(desc)
    if err != nil { return err }
    leafPub := ed25519.PublicKey(bundle.Chain[0].PublicKey)
    if !ed25519.Verify(leafPub, b, bundle.Signature) {
        return errors.New("signature invalid")
    }
    if bundle.KeyID != Fingerprint(leafPub) {
        return errors.New("key id mismatch")
    }
    return nil
}

// SignatureStore stores signature bundles for CIDs.
type SignatureStore interface {
    Put(cid CID, sig SignatureBundle) error
    List(cid CID) ([]SignatureBundle, error)
}

// InMemorySignatureStore is a simple in-memory store for signatures.
type InMemorySignatureStore struct {
    data map[CID][]SignatureBundle
}

func NewInMemorySignatureStore() *InMemorySignatureStore {
    return &InMemorySignatureStore{ data: make(map[CID][]SignatureBundle) }
}

func (s *InMemorySignatureStore) Put(cid CID, sig SignatureBundle) error {
    s.data[cid] = append(s.data[cid], sig)
    return nil
}

func (s *InMemorySignatureStore) List(cid CID) ([]SignatureBundle, error) {
    out := append([]SignatureBundle(nil), s.data[cid]...)
    return out, nil
}

// SignPackage fetches descriptor and stores its signature bundle.
func SignPackage(ctx context.Context, reg Registry, cid CID, priv ed25519.PrivateKey, chain []Certificate, store SignatureStore) (SignatureBundle, error) {
    desc, err := BuildDescriptor(ctx, reg, cid)
    if err != nil { return SignatureBundle{}, err }
    bundle, err := SignDescriptor(desc, priv, chain)
    if err != nil { return SignatureBundle{}, err }
    if err := store.Put(cid, bundle); err != nil { return SignatureBundle{}, err }
    return bundle, nil
}

// VerifyPackage verifies at least one signature for the CID against the trust store.
func VerifyPackage(ctx context.Context, reg Registry, ts *TrustStore, cid CID, store SignatureStore) error {
    desc, err := BuildDescriptor(ctx, reg, cid)
    if err != nil { return err }
    bundles, err := store.List(cid)
    if err != nil { return err }
    if len(bundles) == 0 { return errors.New("no signatures found") }
    var lastErr error
    for _, b := range bundles {
        if err := ts.VerifyDescriptor(desc, b); err == nil { return nil } else { lastErr = err }
    }
    return fmt.Errorf("no valid signatures: last error: %w", lastErr)
}

// VulnerabilityScanner abstracts package vulnerability checks.
type VulnerabilityScanner interface {
    IsVulnerable(desc PackageDescriptor) (bool, string)
}

// InMemoryAdvisoryScanner flags advisories by name@version.
type InMemoryAdvisoryScanner struct { advisories map[string]string }

func NewInMemoryAdvisoryScanner() *InMemoryAdvisoryScanner { return &InMemoryAdvisoryScanner{ advisories: make(map[string]string) } }

func (s *InMemoryAdvisoryScanner) Add(name PackageID, ver Version, reason string) {
    s.advisories[string(name)+"@"+string(ver)] = reason
}

func (s *InMemoryAdvisoryScanner) IsVulnerable(desc PackageDescriptor) (bool, string) {
    key := string(desc.Name)+"@"+string(desc.Version)
    if r, ok := s.advisories[key]; ok { return true, r }
    return false, ""
}

// ValidatePackageSecurity runs both signature verification and vulnerability scan.
func ValidatePackageSecurity(ctx context.Context, reg Registry, ts *TrustStore, cid CID, store SignatureStore, scanner VulnerabilityScanner) error {
    if err := VerifyPackage(ctx, reg, ts, cid, store); err != nil { return err }
    desc, err := BuildDescriptor(ctx, reg, cid)
    if err != nil { return err }
    if ok, reason := scanner.IsVulnerable(desc); ok {
        return fmt.Errorf("package flagged vulnerable: %s", reason)
    }
    return nil
}

func copyStringMap(m map[string]string) map[string]string {
    if m == nil { return nil }
    out := make(map[string]string, len(m))
    for k, v := range m { out[k] = v }
    return out
}

func randomSerial() string {
    var b [16]byte
    _, _ = rand.Read(b[:])
    return hex.EncodeToString(b[:])
}


