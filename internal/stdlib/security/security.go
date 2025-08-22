// Package security provides comprehensive security features including
// access control, authentication, authorization, sandboxing, and security policies.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

// User represents a system user.
type User struct {
	ID            string
	Username      string
	Email         string
	PasswordHash  string
	Salt          string
	Groups        []string
	Permissions   []Permission
	CreatedAt     time.Time
	LastLogin     time.Time
	Enabled       bool
	Locked        bool
	LoginAttempts int
}

// Group represents a user group.
type Group struct {
	ID          string
	Name        string
	Description string
	Permissions []Permission
	Members     []string
	CreatedAt   time.Time
}

// Permission represents a security permission.
type Permission struct {
	ID       string
	Name     string
	Resource string
	Action   Action
	Scope    Scope
}

// Action represents permission actions.
type Action int

const (
	ActionRead Action = iota
	ActionWrite
	ActionExecute
	ActionDelete
	ActionCreate
	ActionModify
	ActionAdmin
)

// Scope represents permission scope.
type Scope int

const (
	ScopeNone Scope = iota
	ScopeOwner
	ScopeGroup
	ScopeAll
	ScopeSystem
)

// Role represents a security role.
type Role struct {
	ID          string
	Name        string
	Description string
	Permissions []Permission
	Inherits    []string
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    string
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
	Active    bool
}

// AccessToken represents an access token.
type AccessToken struct {
	Token     string
	UserID    string
	Scope     []string
	ExpiresAt time.Time
	Type      TokenType
}

// TokenType represents token types.
type TokenType int

const (
	BearerToken TokenType = iota
	APIToken
	RefreshToken
)

// SecurityPolicy represents a security policy.
type SecurityPolicy struct {
	ID          string
	Name        string
	Description string
	Rules       []SecurityRule
	Enabled     bool
	Priority    int
}

// SecurityRule represents a security rule.
type SecurityRule struct {
	ID        string
	Type      RuleType
	Condition string
	Action    RuleAction
	Severity  Severity
}

// RuleType represents rule types.
type RuleType int

const (
	AccessRule RuleType = iota
	AuthenticationRule
	AuthorizationRule
	AuditRule
	ComplianceRule
)

// RuleAction represents rule actions.
type RuleAction int

const (
	Allow RuleAction = iota
	Deny
	Log
	Alert
	Block
)

// Severity represents severity levels.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// Sandbox represents a security sandbox.
type Sandbox struct {
	ID          string
	Name        string
	Permissions []Permission
	Resources   []Resource
	Limits      ResourceLimits
	Policies    []SecurityPolicy
	Active      bool
}

// Resource represents a system resource.
type Resource struct {
	ID   string
	Type ResourceType
	Path string
	Mode ResourceMode
}

// ResourceType represents resource types.
type ResourceType int

const (
	FileResource ResourceType = iota
	NetworkResource
	ProcessResource
	MemoryResource
	DeviceResource
)

// ResourceMode represents resource access modes.
type ResourceMode int

const (
	ReadOnly ResourceMode = iota
	WriteOnly
	ReadWrite
	Execute
	NoAccess
)

// ResourceLimits represents resource usage limits.
type ResourceLimits struct {
	MaxMemory    uint64
	MaxCPU       float64
	MaxFiles     int
	MaxProcesses int
	MaxNetwork   uint64
	MaxDisk      uint64
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID        string
	Timestamp time.Time
	UserID    string
	Action    string
	Resource  string
	Result    AuditResult
	Details   map[string]interface{}
	IPAddress string
	UserAgent string
}

// AuditResult represents audit results.
type AuditResult int

const (
	AuditSuccess AuditResult = iota
	AuditFailure
	AuditDenied
	AuditError
)

// SecurityManager manages security operations.
type SecurityManager struct {
	users     map[string]*User
	groups    map[string]*Group
	roles     map[string]*Role
	sessions  map[string]*Session
	tokens    map[string]*AccessToken
	policies  map[string]*SecurityPolicy
	sandboxes map[string]*Sandbox
	auditLogs []AuditLog
	config    *SecurityConfig
}

// SecurityConfig represents security configuration.
type SecurityConfig struct {
	PasswordMinLength     int
	PasswordRequireUpper  bool
	PasswordRequireLower  bool
	PasswordRequireDigit  bool
	PasswordRequireSymbol bool
	SessionTimeout        time.Duration
	TokenTimeout          time.Duration
	MaxLoginAttempts      int
	LockoutDuration       time.Duration
	TwoFactorRequired     bool
	AuditEnabled          bool
	SandboxEnabled        bool
}

// Default security configuration
var defaultConfig = &SecurityConfig{
	PasswordMinLength:     8,
	PasswordRequireUpper:  true,
	PasswordRequireLower:  true,
	PasswordRequireDigit:  true,
	PasswordRequireSymbol: true,
	SessionTimeout:        24 * time.Hour,
	TokenTimeout:          1 * time.Hour,
	MaxLoginAttempts:      5,
	LockoutDuration:       30 * time.Minute,
	TwoFactorRequired:     false,
	AuditEnabled:          true,
	SandboxEnabled:        true,
}

// Global security manager instance
var securityManager *SecurityManager

// NewSecurityManager creates a new security manager.
func NewSecurityManager(config *SecurityConfig) *SecurityManager {
	if config == nil {
		config = defaultConfig
	}

	return &SecurityManager{
		users:     make(map[string]*User),
		groups:    make(map[string]*Group),
		roles:     make(map[string]*Role),
		sessions:  make(map[string]*Session),
		tokens:    make(map[string]*AccessToken),
		policies:  make(map[string]*SecurityPolicy),
		sandboxes: make(map[string]*Sandbox),
		auditLogs: make([]AuditLog, 0),
		config:    config,
	}
}

// GetSecurityManager returns the global security manager.
func GetSecurityManager() *SecurityManager {
	if securityManager == nil {
		securityManager = NewSecurityManager(nil)
	}
	return securityManager
}

// User management

// CreateUser creates a new user.
func (sm *SecurityManager) CreateUser(username, email, password string) (*User, error) {
	if err := sm.validatePassword(password); err != nil {
		return nil, err
	}

	// Check if user already exists
	for _, user := range sm.users {
		if user.Username == username || user.Email == email {
			return nil, errors.New("user already exists")
		}
	}

	// Generate salt and hash password
	salt, err := generateSalt()
	if err != nil {
		return nil, err
	}

	passwordHash := hashPassword(password, salt)

	user := &User{
		ID:            generateID(),
		Username:      username,
		Email:         email,
		PasswordHash:  passwordHash,
		Salt:          salt,
		Groups:        make([]string, 0),
		Permissions:   make([]Permission, 0),
		CreatedAt:     time.Now(),
		Enabled:       true,
		Locked:        false,
		LoginAttempts: 0,
	}

	sm.users[user.ID] = user
	sm.auditLog(user.ID, "CREATE_USER", "user:"+user.ID, AuditSuccess, nil)

	return user, nil
}

// AuthenticateUser authenticates a user.
func (sm *SecurityManager) AuthenticateUser(username, password string) (*User, error) {
	var user *User

	// Find user by username or email
	for _, u := range sm.users {
		if u.Username == username || u.Email == username {
			user = u
			break
		}
	}

	if user == nil {
		sm.auditLog("", "AUTHENTICATE_USER", "user:"+username, AuditFailure, map[string]interface{}{
			"reason": "user not found",
		})
		return nil, errors.New("invalid credentials")
	}

	// Check if user is locked
	if user.Locked {
		sm.auditLog(user.ID, "AUTHENTICATE_USER", "user:"+user.ID, AuditDenied, map[string]interface{}{
			"reason": "user locked",
		})
		return nil, errors.New("user account locked")
	}

	// Check if user is enabled
	if !user.Enabled {
		sm.auditLog(user.ID, "AUTHENTICATE_USER", "user:"+user.ID, AuditDenied, map[string]interface{}{
			"reason": "user disabled",
		})
		return nil, errors.New("user account disabled")
	}

	// Verify password
	if !verifyPassword(password, user.PasswordHash, user.Salt) {
		user.LoginAttempts++

		// Lock user if too many failed attempts
		if user.LoginAttempts >= sm.config.MaxLoginAttempts {
			user.Locked = true
		}

		sm.auditLog(user.ID, "AUTHENTICATE_USER", "user:"+user.ID, AuditFailure, map[string]interface{}{
			"reason":         "invalid password",
			"login_attempts": user.LoginAttempts,
		})

		return nil, errors.New("invalid credentials")
	}

	// Reset login attempts on successful authentication
	user.LoginAttempts = 0
	user.LastLogin = time.Now()

	sm.auditLog(user.ID, "AUTHENTICATE_USER", "user:"+user.ID, AuditSuccess, nil)

	return user, nil
}

// CreateSession creates a user session.
func (sm *SecurityManager) CreateSession(userID string) (*Session, error) {
	_, exists := sm.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        generateID(),
		UserID:    userID,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sm.config.SessionTimeout),
		Active:    true,
	}

	sm.sessions[session.ID] = session
	sm.auditLog(userID, "CREATE_SESSION", "session:"+session.ID, AuditSuccess, nil)

	return session, nil
}

// ValidateSession validates a session token.
func (sm *SecurityManager) ValidateSession(token string) (*Session, error) {
	for _, session := range sm.sessions {
		if session.Token == token {
			if time.Now().After(session.ExpiresAt) {
				session.Active = false
				sm.auditLog(session.UserID, "VALIDATE_SESSION", "session:"+session.ID, AuditFailure, map[string]interface{}{
					"reason": "session expired",
				})
				return nil, errors.New("session expired")
			}

			if !session.Active {
				sm.auditLog(session.UserID, "VALIDATE_SESSION", "session:"+session.ID, AuditFailure, map[string]interface{}{
					"reason": "session inactive",
				})
				return nil, errors.New("session inactive")
			}

			return session, nil
		}
	}

	sm.auditLog("", "VALIDATE_SESSION", "token:"+token[:8]+"...", AuditFailure, map[string]interface{}{
		"reason": "session not found",
	})
	return nil, errors.New("invalid session")
}

// RevokeSession revokes a session.
func (sm *SecurityManager) RevokeSession(sessionID string) error {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return errors.New("session not found")
	}

	session.Active = false
	delete(sm.sessions, sessionID)

	sm.auditLog(session.UserID, "REVOKE_SESSION", "session:"+sessionID, AuditSuccess, nil)

	return nil
}

// Authorization

// CheckPermission checks if a user has a specific permission.
func (sm *SecurityManager) CheckPermission(userID, resource string, action Action) bool {
	user, exists := sm.users[userID]
	if !exists {
		sm.auditLog(userID, "CHECK_PERMISSION", resource, AuditFailure, map[string]interface{}{
			"reason": "user not found",
			"action": action,
		})
		return false
	}

	// Check user permissions
	for _, perm := range user.Permissions {
		if sm.matchesPermission(perm, resource, action) {
			sm.auditLog(userID, "CHECK_PERMISSION", resource, AuditSuccess, map[string]interface{}{
				"action": action,
				"source": "user",
			})
			return true
		}
	}

	// Check group permissions
	for _, groupID := range user.Groups {
		if group, exists := sm.groups[groupID]; exists {
			for _, perm := range group.Permissions {
				if sm.matchesPermission(perm, resource, action) {
					sm.auditLog(userID, "CHECK_PERMISSION", resource, AuditSuccess, map[string]interface{}{
						"action": action,
						"source": "group",
						"group":  groupID,
					})
					return true
				}
			}
		}
	}

	sm.auditLog(userID, "CHECK_PERMISSION", resource, AuditDenied, map[string]interface{}{
		"action": action,
	})
	return false
}

// GrantPermission grants a permission to a user.
func (sm *SecurityManager) GrantPermission(userID string, permission Permission) error {
	user, exists := sm.users[userID]
	if !exists {
		return errors.New("user not found")
	}

	user.Permissions = append(user.Permissions, permission)

	sm.auditLog(userID, "GRANT_PERMISSION", permission.Resource, AuditSuccess, map[string]interface{}{
		"permission": permission.Name,
		"action":     permission.Action,
	})

	return nil
}

// RevokePermission revokes a permission from a user.
func (sm *SecurityManager) RevokePermission(userID, permissionID string) error {
	user, exists := sm.users[userID]
	if !exists {
		return errors.New("user not found")
	}

	for i, perm := range user.Permissions {
		if perm.ID == permissionID {
			user.Permissions = append(user.Permissions[:i], user.Permissions[i+1:]...)

			sm.auditLog(userID, "REVOKE_PERMISSION", perm.Resource, AuditSuccess, map[string]interface{}{
				"permission": perm.Name,
			})

			return nil
		}
	}

	return errors.New("permission not found")
}

// Group management

// CreateGroup creates a new group.
func (sm *SecurityManager) CreateGroup(name, description string) (*Group, error) {
	group := &Group{
		ID:          generateID(),
		Name:        name,
		Description: description,
		Permissions: make([]Permission, 0),
		Members:     make([]string, 0),
		CreatedAt:   time.Now(),
	}

	sm.groups[group.ID] = group
	sm.auditLog("", "CREATE_GROUP", "group:"+group.ID, AuditSuccess, nil)

	return group, nil
}

// AddUserToGroup adds a user to a group.
func (sm *SecurityManager) AddUserToGroup(userID, groupID string) error {
	user, userExists := sm.users[userID]
	group, groupExists := sm.groups[groupID]

	if !userExists {
		return errors.New("user not found")
	}
	if !groupExists {
		return errors.New("group not found")
	}

	// Check if user is already in group
	for _, id := range user.Groups {
		if id == groupID {
			return errors.New("user already in group")
		}
	}

	user.Groups = append(user.Groups, groupID)
	group.Members = append(group.Members, userID)

	sm.auditLog(userID, "ADD_TO_GROUP", "group:"+groupID, AuditSuccess, nil)

	return nil
}

// Sandbox management

// CreateSandbox creates a new sandbox.
func (sm *SecurityManager) CreateSandbox(name string, limits ResourceLimits) (*Sandbox, error) {
	sandbox := &Sandbox{
		ID:          generateID(),
		Name:        name,
		Permissions: make([]Permission, 0),
		Resources:   make([]Resource, 0),
		Limits:      limits,
		Policies:    make([]SecurityPolicy, 0),
		Active:      true,
	}

	sm.sandboxes[sandbox.ID] = sandbox
	sm.auditLog("", "CREATE_SANDBOX", "sandbox:"+sandbox.ID, AuditSuccess, nil)

	return sandbox, nil
}

// EnterSandbox enters a sandbox environment.
func (sm *SecurityManager) EnterSandbox(sandboxID string) error {
	sandbox, exists := sm.sandboxes[sandboxID]
	if !exists {
		return errors.New("sandbox not found")
	}

	if !sandbox.Active {
		return errors.New("sandbox not active")
	}

	// Apply sandbox restrictions
	// In a real implementation, this would configure the runtime environment

	sm.auditLog("", "ENTER_SANDBOX", "sandbox:"+sandboxID, AuditSuccess, nil)

	return nil
}

// Security policies

// CreatePolicy creates a security policy.
func (sm *SecurityManager) CreatePolicy(name, description string, rules []SecurityRule) (*SecurityPolicy, error) {
	policy := &SecurityPolicy{
		ID:          generateID(),
		Name:        name,
		Description: description,
		Rules:       rules,
		Enabled:     true,
		Priority:    1,
	}

	sm.policies[policy.ID] = policy
	sm.auditLog("", "CREATE_POLICY", "policy:"+policy.ID, AuditSuccess, nil)

	return policy, nil
}

// EvaluatePolicy evaluates security policies.
func (sm *SecurityManager) EvaluatePolicy(userID, resource, action string) bool {
	for _, policy := range sm.policies {
		if !policy.Enabled {
			continue
		}

		for _, rule := range policy.Rules {
			if sm.evaluateRule(rule, userID, resource, action) {
				switch rule.Action {
				case Deny, Block:
					sm.auditLog(userID, "POLICY_VIOLATION", resource, AuditDenied, map[string]interface{}{
						"policy": policy.Name,
						"rule":   rule.ID,
						"action": action,
					})
					return false
				case Log, Alert:
					sm.auditLog(userID, "POLICY_ALERT", resource, AuditSuccess, map[string]interface{}{
						"policy": policy.Name,
						"rule":   rule.ID,
						"action": action,
					})
				}
			}
		}
	}

	return true
}

// Helper methods

func (sm *SecurityManager) matchesPermission(perm Permission, resource string, action Action) bool {
	// Simple wildcard matching
	if perm.Resource == "*" || perm.Resource == resource {
		return perm.Action == action || perm.Action == ActionAdmin
	}

	// Pattern matching
	if strings.HasSuffix(perm.Resource, "*") {
		prefix := strings.TrimSuffix(perm.Resource, "*")
		if strings.HasPrefix(resource, prefix) {
			return perm.Action == action || perm.Action == ActionAdmin
		}
	}

	return false
}

func (sm *SecurityManager) evaluateRule(rule SecurityRule, userID, resource, action string) bool {
	// Simple condition evaluation
	// In a real implementation, this would be more sophisticated
	return strings.Contains(rule.Condition, resource) || strings.Contains(rule.Condition, action)
}

func (sm *SecurityManager) validatePassword(password string) error {
	if len(password) < sm.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", sm.config.PasswordMinLength)
	}

	if sm.config.PasswordRequireUpper && !containsUpper(password) {
		return errors.New("password must contain uppercase letter")
	}

	if sm.config.PasswordRequireLower && !containsLower(password) {
		return errors.New("password must contain lowercase letter")
	}

	if sm.config.PasswordRequireDigit && !containsDigit(password) {
		return errors.New("password must contain digit")
	}

	if sm.config.PasswordRequireSymbol && !containsSymbol(password) {
		return errors.New("password must contain symbol")
	}

	return nil
}

func (sm *SecurityManager) auditLog(userID, action, resource string, result AuditResult, details map[string]interface{}) {
	if !sm.config.AuditEnabled {
		return
	}

	log := AuditLog{
		ID:        generateID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Result:    result,
		Details:   details,
	}

	sm.auditLogs = append(sm.auditLogs, log)
}

// Utility functions

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func generateSalt() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

func hashPassword(password, salt string) string {
	hash := sha256.Sum256([]byte(password + salt))
	return fmt.Sprintf("%x", hash)
}

func verifyPassword(password, hash, salt string) bool {
	return hashPassword(password, salt) == hash
}

func containsUpper(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

func containsLower(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func containsSymbol(s string) bool {
	symbols := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, c := range s {
		for _, sym := range symbols {
			if c == sym {
				return true
			}
		}
	}
	return false
}

// Public API functions

// Initialize initializes the security system.
func Initialize(config *SecurityConfig) {
	securityManager = NewSecurityManager(config)
}

// CreateUser creates a new user.
func CreateUser(username, email, password string) (*User, error) {
	return GetSecurityManager().CreateUser(username, email, password)
}

// Login authenticates a user and creates a session.
func Login(username, password string) (*Session, error) {
	sm := GetSecurityManager()
	user, err := sm.AuthenticateUser(username, password)
	if err != nil {
		return nil, err
	}

	return sm.CreateSession(user.ID)
}

// Logout revokes a session.
func Logout(sessionID string) error {
	return GetSecurityManager().RevokeSession(sessionID)
}

// CheckAccess checks if a user has access to a resource.
func CheckAccess(userID, resource string, action Action) bool {
	sm := GetSecurityManager()
	return sm.CheckPermission(userID, resource, action) && sm.EvaluatePolicy(userID, resource, fmt.Sprintf("%d", action))
}

// GetAuditLogs returns audit logs.
func GetAuditLogs() []AuditLog {
	return GetSecurityManager().auditLogs
}

// Secure function decorator
func Secure(userID, resource string, action Action, fn func() error) error {
	if !CheckAccess(userID, resource, action) {
		return errors.New("access denied")
	}

	return fn()
}
