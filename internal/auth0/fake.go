package auth0

import "context"

// FakeClient is a test implementation of Client
type FakeClient struct {
	Users map[string]*UserInfo // keyed by access token
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		Users: make(map[string]*UserInfo),
	}
}

func (c *FakeClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if user, ok := c.Users[accessToken]; ok {
		return user, nil
	}
	return nil, ErrUserInfoFailed
}

// AddUser adds a user to the fake for testing
func (c *FakeClient) AddUser(accessToken string, info *UserInfo) {
	c.Users[accessToken] = info
}
