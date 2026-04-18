package jwt

import "time"

// NewServiceFromSharedOrSplit issues/verifies tokens. If both accessSecret and refreshSecret are non-empty,
// they are used; otherwise sharedSecret is used for both (dev / simple deploys).
func NewServiceFromSharedOrSplit(sharedSecret, accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer, audience string) (*Service, error) {
	opt := ServiceOptions{
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
		Issuer:     issuer,
		Audience:   audience,
	}
	if accessSecret != "" && refreshSecret != "" {
		opt.AccessSecret = accessSecret
		opt.RefreshSecret = refreshSecret
	} else {
		opt.SharedSecret = sharedSecret
	}
	return NewServiceOptions(opt)
}
