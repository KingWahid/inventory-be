package jwt

// Parse parses any valid token type (access or refresh).
func Parse(svc *Service, token string) (*Claims, error) {
	return svc.Parse(token)
}

// ParseAccess parses access token and enforces token_type=access.
func ParseAccess(svc *Service, token string) (*Claims, error) {
	return svc.ParseAccess(token)
}

// ParseRefresh parses refresh token and enforces token_type=refresh.
func ParseRefresh(svc *Service, token string) (*Claims, error) {
	return svc.ParseRefresh(token)
}
