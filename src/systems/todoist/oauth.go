package todoist

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"orivo/src/systems/files"
)

const (
	clientName = "orivo"
	scope      = "data:read"
	tokenEnv   = "TODOIST_API_TOKEN"
	expirySlop = time.Minute
)

var ErrNotConnected = errors.New("not connected to Todoist")

type OAuth struct {
	RegisterURL  string
	AuthorizeURL string
	TokenURL     string
	HTTP         *http.Client
}

func NewOAuth() *OAuth {
	return &OAuth{
		RegisterURL:  "https://api.todoist.com/oauth/register",
		AuthorizeURL: "https://app.todoist.com/oauth/authorize",
		TokenURL:     "https://api.todoist.com/oauth/access_token",
		HTTP:         &http.Client{Timeout: 20 * time.Second},
	}
}

type Credentials struct {
	ClientID     string    `json:"client_id,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitzero"`
}

func (c Credentials) expired(now time.Time) bool {
	return !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt.Add(-expirySlop))
}

func (o *OAuth) Register(redirectURI string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"client_name":                clientName,
		"redirect_uris":              []string{redirectURI},
		"scope":                      scope,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_method": "none",
	})
	if err != nil {
		return "", err
	}

	resp, err := o.HTTP.Post(o.RegisterURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Todoist refused to register the app: %s", resp.Status)
	}

	var registered struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registered); err != nil {
		return "", err
	}
	if registered.ClientID == "" {
		return "", errors.New("Todoist returned no client id")
	}
	return registered.ClientID, nil
}

func (o *OAuth) LoginURL(clientID, redirectURI, state, challenge string) string {
	query := url.Values{
		"client_id":             {clientID},
		"scope":                 {scope},
		"state":                 {state},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return o.AuthorizeURL + "?" + query.Encode()
}

func (o *OAuth) Exchange(clientID, redirectURI, code, verifier string) (Credentials, error) {
	return o.token(clientID, url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	})
}

func (o *OAuth) Refresh(creds Credentials) (Credentials, error) {
	return o.token(creds.ClientID, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {creds.ClientID},
		"refresh_token": {creds.RefreshToken},
	})
}

func (o *OAuth) token(clientID string, form url.Values) (Credentials, error) {
	resp, err := o.HTTP.PostForm(o.TokenURL, form)
	if err != nil {
		return Credentials{}, err
	}
	defer resp.Body.Close()

	var issued struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&issued); err != nil && resp.StatusCode == http.StatusOK {
		return Credentials{}, err
	}

	switch {
	case issued.Error == "invalid_grant":
		return Credentials{}, ErrTokenRejected
	case resp.StatusCode != http.StatusOK || issued.AccessToken == "":
		return Credentials{}, fmt.Errorf("Todoist refused to issue a token: %s %s", resp.Status, issued.Error)
	}

	creds := Credentials{
		ClientID:     clientID,
		AccessToken:  issued.AccessToken,
		RefreshToken: issued.RefreshToken,
	}
	if issued.ExpiresIn > 0 {
		creds.ExpiresAt = time.Now().Add(time.Duration(issued.ExpiresIn) * time.Second).Round(0)
	}
	return creds, nil
}

func NewState() string {
	return randomString()
}

func NewPKCE() (verifier, challenge string) {
	verifier = randomString()
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomString() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func LoadCredentials(path string) (Credentials, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, ErrNotConnected
	}
	if err != nil {
		return Credentials{}, err
	}

	var creds Credentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return Credentials{}, err
	}
	if creds.AccessToken == "" {
		return Credentials{}, ErrNotConnected
	}
	return creds, nil
}

func SaveCredentials(path string, creds Credentials) error {
	raw, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return files.WriteAtomic(path, raw)
}

func (o *OAuth) Token(path string) (string, error) {
	if token := strings.TrimSpace(os.Getenv(tokenEnv)); token != "" {
		return token, nil
	}

	creds, err := LoadCredentials(path)
	if err != nil {
		return "", err
	}
	if !creds.expired(time.Now()) {
		return creds.AccessToken, nil
	}
	if creds.RefreshToken == "" {
		return "", ErrTokenRejected
	}

	creds, err = o.Refresh(creds)
	if err != nil {
		return "", err
	}
	if err := SaveCredentials(path, creds); err != nil {
		return "", err
	}
	return creds.AccessToken, nil
}
