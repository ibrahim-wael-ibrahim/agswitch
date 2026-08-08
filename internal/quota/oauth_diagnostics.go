package quota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

const accessTokenInfoURL = "https://www.googleapis.com/oauth2/v2/tokeninfo"

type OAuthTokenInfo struct {
	IssuedTo  string   `json:"issued_to,omitempty"`
	Audience  string   `json:"audience,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresIn int      `json:"expires_in,omitempty"`
}

type AuthDiagnostic struct {
	Profile string `json:"profile"`
	Email   string `json:"email,omitempty"`
	Active  bool   `json:"active,omitempty"`

	AccessTokenPresent  bool `json:"access_token_present"`
	RefreshTokenPresent bool `json:"refresh_token_present"`

	ClientIDSource         string `json:"client_id_source"`
	EffectiveClientID      string `json:"effective_client_id,omitempty"`
	ClientSecretConfigured bool   `json:"client_secret_configured"`

	CurrentTokenInfo      *OAuthTokenInfo `json:"current_token_info,omitempty"`
	CurrentTokenInfoError string          `json:"current_token_info_error,omitempty"`
	ClientIDMatchesToken  *bool           `json:"client_id_matches_token,omitempty"`

	RefreshStatus           string `json:"refresh_status"`
	RefreshHTTPStatus       int    `json:"refresh_http_status,omitempty"`
	RefreshError            string `json:"refresh_error,omitempty"`
	RefreshErrorDescription string `json:"refresh_error_description,omitempty"`
	RefreshErrorSubtype     string `json:"refresh_error_subtype,omitempty"`

	RefreshedTokenInfo *OAuthTokenInfo `json:"refreshed_token_info,omitempty"`

	QuotaStatus string `json:"quota_status,omitempty"`
	QuotaError  string `json:"quota_error,omitempty"`
	QuotaModels int    `json:"quota_models,omitempty"`
	QuotaSource string `json:"quota_source,omitempty"`

	Error string `json:"error,omitempty"`
}

type authDiagnoser interface {
	DiagnoseAuth(context.Context, credentials.Credential) AuthDiagnostic
}

// DiagnoseAuth performs a read-only authentication check for one saved profile.
// It never returns access tokens, refresh tokens, or the client secret. A
// successful refresh is used only in memory for tokeninfo/quota validation.
func (s *Service) DiagnoseAuth(ctx context.Context, item account.Account) AuthDiagnostic {
	result := AuthDiagnostic{Profile: item.ID, Email: item.Email, Active: item.Active}
	if s == nil || s.Profiles == nil {
		result.Error = "quota profile store is not configured"
		return result
	}
	credential, err := s.Profiles.Load(ctx, item.ID)
	if err != nil {
		result.Error = "load saved credential: " + err.Error()
		return result
	}
	if result.Email == "" {
		result.Email = credential.Email
	}
	diagnoser, ok := s.Provider.(authDiagnoser)
	if !ok {
		result.Error = "quota provider does not support OAuth diagnostics"
		return result
	}
	diagnosed := diagnoser.DiagnoseAuth(ctx, credential)
	diagnosed.Profile = item.ID
	diagnosed.Active = item.Active
	if diagnosed.Email == "" {
		diagnosed.Email = result.Email
	}
	return diagnosed
}

func (s *Service) DiagnoseAllAuth(ctx context.Context, accounts []account.Account) []AuthDiagnostic {
	results := make([]AuthDiagnostic, len(accounts))
	limit := s.Concurrency
	if limit <= 0 {
		limit = 4
	}
	semaphore := make(chan struct{}, limit)
	var wait sync.WaitGroup
	for index, item := range accounts {
		index, item := index, item
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = AuthDiagnostic{Profile: item.ID, Email: item.Email, Active: item.Active, Error: ctx.Err().Error()}
				return
			}
			results[index] = s.DiagnoseAuth(ctx, item)
		}()
	}
	wait.Wait()
	return results
}

func (p *GoogleProvider) DiagnoseAuth(ctx context.Context, credential credentials.Credential) AuthDiagnostic {
	diagnostic := AuthDiagnostic{Email: credential.Email, RefreshStatus: "not_attempted"}
	auth, err := ExtractAuth(credential.Raw)
	if err != nil {
		diagnostic.Error = "extract authentication: " + err.Error()
		return diagnostic
	}
	diagnostic.AccessTokenPresent = strings.TrimSpace(auth.AccessToken) != ""
	diagnostic.RefreshTokenPresent = strings.TrimSpace(auth.RefreshToken) != ""

	credentialClientID := strings.TrimSpace(auth.ClientID)
	credentialClientSecret := strings.TrimSpace(auth.ClientSecret)
	if credentialClientID != "" {
		diagnostic.ClientIDSource = "credential"
		diagnostic.EffectiveClientID = credentialClientID
	} else if configured := strings.TrimSpace(p.ClientID); configured != "" {
		diagnostic.ClientIDSource = "environment"
		diagnostic.EffectiveClientID = configured
	} else {
		diagnostic.ClientIDSource = "missing"
	}
	diagnostic.ClientSecretConfigured = credentialClientSecret != "" || strings.TrimSpace(p.ClientSecret) != ""

	if diagnostic.AccessTokenPresent {
		info, infoErr := p.inspectAccessToken(ctx, auth.AccessToken)
		if infoErr != nil {
			diagnostic.CurrentTokenInfoError = safeDiagnosticError(infoErr)
		} else {
			diagnostic.CurrentTokenInfo = &info
			diagnostic.ClientIDMatchesToken = compareClientID(diagnostic.EffectiveClientID, info)
		}
	}

	if !diagnostic.RefreshTokenPresent {
		diagnostic.RefreshStatus = "missing_refresh_token"
		if diagnostic.AccessTokenPresent {
			p.testQuotaAccess(ctx, auth.AccessToken, auth.ProjectID, &diagnostic)
		}
		return diagnostic
	}
	if diagnostic.EffectiveClientID == "" {
		diagnostic.RefreshStatus = "missing_client_id"
		return diagnostic
	}

	refreshAuth := auth
	refreshAuth.ClientID = diagnostic.EffectiveClientID
	if strings.TrimSpace(refreshAuth.ClientSecret) == "" {
		refreshAuth.ClientSecret = strings.TrimSpace(p.ClientSecret)
	}
	refreshedToken, failure, refreshErr := p.refreshForDiagnostic(ctx, refreshAuth)
	if refreshErr != nil {
		diagnostic.RefreshStatus = "failed"
		diagnostic.RefreshError = safeDiagnosticError(refreshErr)
		if failure != nil {
			diagnostic.RefreshHTTPStatus = failure.Status
			diagnostic.RefreshError = failure.Code
			diagnostic.RefreshErrorDescription = failure.Description
			diagnostic.RefreshErrorSubtype = failure.Subtype
		}
		if diagnostic.AccessTokenPresent {
			p.testQuotaAccess(ctx, auth.AccessToken, auth.ProjectID, &diagnostic)
		}
		return diagnostic
	}

	diagnostic.RefreshStatus = "ok"
	if info, infoErr := p.inspectAccessToken(ctx, refreshedToken); infoErr == nil {
		diagnostic.RefreshedTokenInfo = &info
		if diagnostic.ClientIDMatchesToken == nil {
			diagnostic.ClientIDMatchesToken = compareClientID(diagnostic.EffectiveClientID, info)
		}
	}
	p.testQuotaAccess(ctx, refreshedToken, auth.ProjectID, &diagnostic)
	return diagnostic
}

type oauthFailure struct {
	Status      int
	Code        string
	Description string
	Subtype     string
}

func (p *GoogleProvider) refreshForDiagnostic(ctx context.Context, auth AuthMaterial) (string, *oauthFailure, error) {
	values := url.Values{
		"client_id":     {strings.TrimSpace(auth.ClientID)},
		"refresh_token": {strings.TrimSpace(auth.RefreshToken)},
		"grant_type":    {"refresh_token"},
	}
	if secret := strings.TrimSpace(auth.ClientSecret); secret != "" {
		values.Set("client_secret", secret)
	}
	tokenURL := strings.TrimSpace(p.TokenURL)
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.httpClient().Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("refresh access token: %w", err)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if readErr != nil {
		return "", nil, fmt.Errorf("read OAuth response: %w", readErr)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		failure := parseOAuthFailure(response.StatusCode, body)
		return "", &failure, errors.New(failure.Error())
	}
	var output struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &output); err != nil {
		return "", nil, fmt.Errorf("decode refreshed access token: %w", err)
	}
	if strings.TrimSpace(output.AccessToken) == "" {
		return "", nil, errors.New("token refresh returned an empty access token")
	}
	return strings.TrimSpace(output.AccessToken), nil, nil
}

func (f oauthFailure) Error() string {
	if f.Code != "" && f.Description != "" {
		return fmt.Sprintf("OAuth %s (HTTP %d): %s", f.Code, f.Status, f.Description)
	}
	if f.Code != "" {
		return fmt.Sprintf("OAuth %s (HTTP %d)", f.Code, f.Status)
	}
	return fmt.Sprintf("OAuth refresh failed (HTTP %d)", f.Status)
}

func parseOAuthFailure(status int, body []byte) oauthFailure {
	failure := oauthFailure{Status: status}
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		ErrorSubtype     string `json:"error_subtype"`
	}
	if json.Unmarshal(body, &payload) == nil {
		failure.Code = sanitizeOAuthField(payload.Error)
		failure.Description = sanitizeOAuthField(payload.ErrorDescription)
		failure.Subtype = sanitizeOAuthField(payload.ErrorSubtype)
	}
	return failure
}

func sanitizeOAuthField(value string) string {
	value = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, value))
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 320 {
		value = value[:320] + "…"
	}
	return value
}

func (p *GoogleProvider) inspectAccessToken(ctx context.Context, token string) (OAuthTokenInfo, error) {
	var output OAuthTokenInfo
	endpoint, err := url.Parse(accessTokenInfoURL)
	if err != nil {
		return output, err
	}
	query := endpoint.Query()
	query.Set("access_token", token)
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return output, err
	}
	response, err := p.httpClient().Do(req)
	if err != nil {
		return output, fmt.Errorf("tokeninfo request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return output, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return output, fmt.Errorf("tokeninfo HTTP %d", response.StatusCode)
	}
	var payload struct {
		IssuedTo  string `json:"issued_to"`
		Audience  string `json:"audience"`
		Scope     string `json:"scope"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return output, fmt.Errorf("decode tokeninfo: %w", err)
	}
	output.IssuedTo = strings.TrimSpace(payload.IssuedTo)
	output.Audience = strings.TrimSpace(payload.Audience)
	output.ExpiresIn = payload.ExpiresIn
	if scope := strings.TrimSpace(payload.Scope); scope != "" {
		output.Scopes = strings.Fields(scope)
	}
	return output, nil
}

func compareClientID(clientID string, info OAuthTokenInfo) *bool {
	clientID = strings.TrimSpace(clientID)
	issued := strings.TrimSpace(info.IssuedTo)
	if issued == "" {
		issued = strings.TrimSpace(info.Audience)
	}
	if clientID == "" || issued == "" {
		return nil
	}
	matches := clientID == issued
	return &matches
}

func (p *GoogleProvider) testQuotaAccess(ctx context.Context, token, projectID string, diagnostic *AuthDiagnostic) {
	if strings.TrimSpace(token) == "" || diagnostic == nil {
		return
	}
	if strings.TrimSpace(projectID) == "" {
		if load, err := p.loadCodeAssist(ctx, token); err == nil {
			projectID = load.projectID()
		}
	}
	models, source, err := p.fetchAvailableModels(ctx, token, projectID)
	if err != nil {
		diagnostic.QuotaStatus = "failed"
		diagnostic.QuotaError = safeDiagnosticError(err)
		return
	}
	diagnostic.QuotaStatus = "ok"
	diagnostic.QuotaModels = len(models.Models)
	diagnostic.QuotaSource = source
}

func safeDiagnosticError(err error) string {
	if err == nil {
		return ""
	}
	return sanitizeOAuthField(err.Error())
}
