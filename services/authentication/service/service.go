package service

import "context"

// RegisterInput contains payload for creating tenant and first admin.
type RegisterInput struct {
	TenantName string
	AdminName  string
	AdminEmail string
	Password   string
}

// RegisterResult contains identifiers for successful registration.
type RegisterResult struct {
	TenantID string
	UserID   string
	Email    string
}

// LoginInput is credential payload for authentication.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult is authentication success payload.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

// RefreshInput carries the refresh JWT from the client.
type RefreshInput struct {
	RefreshToken string
}

// MeResult is the current user profile for GET /auth/me.
type MeResult struct {
	UserID   string
	TenantID string
	Email    string
	FullName string
}

// Service is the authentication application facade.
type Service interface {
	PingDB(ctx context.Context) error
	RegisterTenantAdmin(ctx context.Context, in RegisterInput) (RegisterResult, error)
	Login(ctx context.Context, in LoginInput) (LoginResult, error)
	Refresh(ctx context.Context, in RefreshInput) (LoginResult, error)
	Logout(ctx context.Context) error
	Me(ctx context.Context) (MeResult, error)
}
