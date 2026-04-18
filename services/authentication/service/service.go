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

// Service is the authentication application facade.
type Service interface {
	PingDB(ctx context.Context) error
	RegisterTenantAdmin(ctx context.Context, in RegisterInput) (RegisterResult, error)
	Login(ctx context.Context, in LoginInput) (LoginResult, error)
}
