package jira

import "context"

// NewTokenStore creates a new TokenStore (stub implementation).
func NewTokenStore(db interface{}) TokenStore {
	return &stubTokenStore{}
}

type stubTokenStore struct{}

func (s *stubTokenStore) GetTokens(ctx context.Context, orgID uint64) (*OAuthTokens, error) {
	// TODO: implement real database lookup
	return &OAuthTokens{}, nil
}

func (s *stubTokenStore) SaveTokens(ctx context.Context, orgID uint64, tokens *OAuthTokens) error {
	// TODO: implement real database save
	return nil
}
