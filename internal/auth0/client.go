package auth0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrUserInfoFailed = errors.New("failed to fetch user info")

// UserInfo represents the response from Auth0's /userinfo endpoint
type UserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Nickname      string `json:"nickname"`
	Picture       string `json:"picture"`
}

// Client is an interface for Auth0 API operations
type Client interface {
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

// HTTPClient implements Client using real HTTP calls
type HTTPClient struct {
	domain     string
	httpClient *http.Client
}

func NewHTTPClient(domain string) *HTTPClient {
	return &HTTPClient{
		domain: domain,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	url := fmt.Sprintf("https://%s/userinfo", c.domain)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfoFailed, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfoFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUserInfoFailed, resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfoFailed, err)
	}

	return &userInfo, nil
}
