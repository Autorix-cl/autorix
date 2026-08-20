package autorix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// EgoClient provides user identity, session management, and traits operations.
type EgoClient struct {
	client *Client
}

func newEgoClient(c *Client) *EgoClient {
	return &EgoClient{client: c}
}

// UserSession describes an active session returned by Ego.
type UserSession struct {
	ID        string `json:"id"`
	Active    bool   `json:"active"`
	Identity  User   `json:"identity"`
	ExpiresAt string `json:"expires_at"`
}

// WhoAmI validates a session token or cookie and returns the active identity.
func (e *EgoClient) WhoAmI(ctx context.Context, sessionToken string) (*UserSession, error) {
	url := strings.TrimRight(e.client.config.EgoURL, "/") + "/sessions/whoami"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if sessionToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+sessionToken)
		httpReq.AddCookie(&http.Cookie{
			Name:  "autorix_session_token",
			Value: sessionToken,
		})
	}
	e.client.prepareRequest(ctx, httpReq)

	httpResp, err := executeWithRetry(ctx, e.client.config.RetryConfig, func(opCtx context.Context) (*http.Response, error) {
		return e.client.config.HTTPClient.Do(httpReq)
	})

	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session invalid or expired (status: %d)", httpResp.StatusCode)
	}

	var session UserSession
	if err := json.NewDecoder(httpResp.Body).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}
