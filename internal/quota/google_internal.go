package quota

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

const (
	defaultTokenURL = "https://oauth2.googleapis.com/token"
	userAgent       = "antigravity"
)

var defaultBaseURLs = []string{
	"https://cloudcode-pa.googleapis.com",
	"https://daily-cloudcode-pa.sandbox.googleapis.com",
	"https://autopush-cloudcode-pa.sandbox.googleapis.com",
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type GoogleProvider struct {
	HTTP         HTTPDoer
	BaseURLs     []string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Now          func() time.Time
}

func NewGoogleProvider() *GoogleProvider {
	return &GoogleProvider{
		HTTP:         &http.Client{Timeout: 15 * time.Second},
		BaseURLs:     append([]string(nil), defaultBaseURLs...),
		TokenURL:     defaultTokenURL,
		ClientID:     strings.TrimSpace(os.Getenv("AGSWITCH_OAUTH_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("AGSWITCH_OAUTH_CLIENT_SECRET")),
		Now:          time.Now,
	}
}

func (p *GoogleProvider) Fetch(ctx context.Context, credential credentials.Credential) (Snapshot, error) {
	auth, err := ExtractAuth(credential.Raw)
	if err != nil {
		return Snapshot{}, fmt.Errorf("extract quota authentication: %w", err)
	}
	if auth.ClientID == "" {
		auth.ClientID = strings.TrimSpace(p.ClientID)
	}
	if auth.ClientSecret == "" {
		auth.ClientSecret = strings.TrimSpace(p.ClientSecret)
	}

	token := auth.AccessToken
	if token == "" {
		token, err = p.refreshAccessToken(ctx, auth)
		if err != nil {
			return Snapshot{}, err
		}
	}

	projectID := auth.ProjectID
	tier := ""
	loadResponse, loadErr := p.loadCodeAssist(ctx, token)
	if isUnauthorized(loadErr) && auth.RefreshToken != "" {
		token, err = p.refreshAccessToken(ctx, auth)
		if err != nil {
			return Snapshot{}, err
		}
		loadResponse, loadErr = p.loadCodeAssist(ctx, token)
	}
	if loadErr == nil {
		if projectID == "" {
			projectID = loadResponse.projectID()
		}
		tier = loadResponse.tier()
	}

	modelsResponse, endpoint, err := p.fetchAvailableModels(ctx, token, projectID)
	if isUnauthorized(err) && auth.RefreshToken != "" {
		token, refreshErr := p.refreshAccessToken(ctx, auth)
		if refreshErr != nil {
			return Snapshot{}, refreshErr
		}
		modelsResponse, endpoint, err = p.fetchAvailableModels(ctx, token, projectID)
	}
	if err != nil {
		return Snapshot{}, err
	}
	if len(modelsResponse.Models) == 0 {
		return Snapshot{}, errors.New("quota API returned no models")
	}

	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now().UTC()
	}
	snapshot := Snapshot{
		Email:            credential.Email,
		SubscriptionTier: tier,
		Models:           make(map[string]ModelUsage, len(modelsResponse.Models)),
		FetchedAt:        now,
		Source:           "google-cloud-code",
		Metadata:         map[string]string{"endpoint": endpoint},
	}
	for id, model := range modelsResponse.Models {
		name := strings.TrimSpace(model.DisplayName)
		if name == "" {
			name = strings.TrimSpace(model.Label)
		}
		if name == "" {
			name = id
		}
		remaining := -1
		if model.QuotaInfo.RemainingFraction != nil {
			fraction := math.Max(0, math.Min(1, *model.QuotaInfo.RemainingFraction))
			remaining = int(math.Round(fraction * 100))
		}
		if model.QuotaInfo.IsExhausted {
			remaining = 0
		}
		var resetAt time.Time
		if value := strings.TrimSpace(model.QuotaInfo.ResetTime); value != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, value); parseErr == nil {
				resetAt = parsed
			}
		}
		snapshot.Models[id] = ModelUsage{
			ID:        id,
			Name:      name,
			Remaining: remaining,
			Limit:     100,
			ResetAt:   resetAt,
			Exhausted: model.QuotaInfo.IsExhausted || remaining == 0,
		}
	}
	return snapshot, nil
}

type loadCodeAssistResponse struct {
	PlanInfo struct {
		PlanType string `json:"planType"`
	} `json:"planInfo"`
	CloudProject json.RawMessage `json:"cloudaicompanionProject"`
	CurrentTier  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"currentTier"`
	PaidTier struct {
		ID string `json:"id"`
	} `json:"paidTier"`
}

func (r loadCodeAssistResponse) projectID() string {
	if len(r.CloudProject) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(r.CloudProject, &text) == nil {
		return strings.TrimSpace(text)
	}
	var object struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(r.CloudProject, &object) == nil {
		return strings.TrimSpace(object.ID)
	}
	return ""
}

func (r loadCodeAssistResponse) tier() string {
	for _, value := range []string{r.CurrentTier.Name, r.CurrentTier.ID, r.PlanInfo.PlanType, r.PaidTier.ID} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

type fetchModelsResponse struct {
	Models map[string]struct {
		DisplayName string `json:"displayName"`
		Label       string `json:"label"`
		QuotaInfo   struct {
			RemainingFraction *float64 `json:"remainingFraction"`
			ResetTime         string   `json:"resetTime"`
			IsExhausted       bool     `json:"isExhausted"`
		} `json:"quotaInfo"`
	} `json:"models"`
}

func (p *GoogleProvider) loadCodeAssist(ctx context.Context, token string) (loadCodeAssistResponse, error) {
	var output loadCodeAssistResponse
	_, err := p.postWithFallback(ctx, token, "/v1internal:loadCodeAssist", map[string]any{
		"metadata": map[string]string{
			"ideType":    "ANTIGRAVITY",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}, &output)
	return output, err
}

func (p *GoogleProvider) fetchAvailableModels(ctx context.Context, token, projectID string) (fetchModelsResponse, string, error) {
	body := map[string]string{}
	if strings.TrimSpace(projectID) != "" {
		body["project"] = strings.TrimSpace(projectID)
	}
	var output fetchModelsResponse
	endpoint, err := p.postWithFallback(ctx, token, "/v1internal:fetchAvailableModels", body, &output)
	return output, endpoint, err
}

func (p *GoogleProvider) postWithFallback(ctx context.Context, token, path string, body any, output any) (string, error) {
	bases := p.BaseURLs
	if len(bases) == 0 {
		bases = defaultBaseURLs
	}
	var last error
	for _, base := range bases {
		endpoint := strings.TrimRight(base, "/") + path
		err := p.postJSON(ctx, endpoint, token, body, output)
		if err == nil {
			return strings.TrimRight(base, "/"), nil
		}
		last = err
		var statusErr *statusError
		if !errors.As(err, &statusErr) || (statusErr.Status != http.StatusTooManyRequests && statusErr.Status < 500) {
			return "", err
		}
	}
	if last == nil {
		last = errors.New("quota API endpoints are not configured")
	}
	return "", last
}

func (p *GoogleProvider) postJSON(ctx context.Context, endpoint, token string, body any, output any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	response, err := p.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("quota API request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return &statusError{Status: response.StatusCode}
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("decode quota API response: %w", err)
	}
	return nil
}

func (p *GoogleProvider) refreshAccessToken(ctx context.Context, auth AuthMaterial) (string, error) {
	if strings.TrimSpace(auth.RefreshToken) == "" {
		return "", errors.New("access token expired and no refresh token is available")
	}
	if strings.TrimSpace(auth.ClientID) == "" {
		return "", errors.New("access token refresh requires client_id in the credential or AGSWITCH_OAUTH_CLIENT_ID")
	}
	token, failure, err := p.refreshForDiagnostic(ctx, auth)
	if err == nil {
		return token, nil
	}
	if failure != nil {
		return "", errors.New(failure.Error())
	}
	return "", fmt.Errorf("refresh access token: %w", err)
}

func (p *GoogleProvider) httpClient() HTTPDoer {
	if p.HTTP != nil {
		return p.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}

type statusError struct {
	Status int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("quota API returned HTTP %d", e.Status)
}

func isUnauthorized(err error) bool {
	var statusErr *statusError
	return errors.As(err, &statusErr) && (statusErr.Status == http.StatusUnauthorized || statusErr.Status == http.StatusForbidden)
}
