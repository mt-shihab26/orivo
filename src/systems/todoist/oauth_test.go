package todoist

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"
)

func fakeOAuth(t *testing.T, handler http.HandlerFunc) *OAuth {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	oauth := NewOAuth()
	oauth.RegisterURL = server.URL + "/register"
	oauth.AuthorizeURL = server.URL + "/authorize"
	oauth.TokenURL = server.URL + "/token"
	return oauth
}

func TestRegisterAsksForAPublicClientWithRefresh(t *testing.T) {
	oauth := fakeOAuth(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RedirectURIs []string `json:"redirect_uris"`
			Scope        string   `json:"scope"`
			GrantTypes   []string `json:"grant_types"`
			AuthMethod   string   `json:"token_endpoint_auth_method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.RedirectURIs) != 1 || body.RedirectURIs[0] != "http://localhost:1/callback" {
			t.Errorf("redirect uris = %v", body.RedirectURIs)
		}
		if body.Scope != "data:read" || body.AuthMethod != "none" || len(body.GrantTypes) != 2 {
			t.Errorf("registration = %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"client_id": "tdd_abc"}`))
	})

	clientID, err := oauth.Register("http://localhost:1/callback")
	if err != nil || clientID != "tdd_abc" {
		t.Fatalf("client id = %q, err = %v", clientID, err)
	}
}

func TestLoginURLCarriesStateAndChallenge(t *testing.T) {
	verifier, challenge := NewPKCE()
	sum := sha256.Sum256([]byte(verifier))
	if challenge != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Fatalf("challenge is not the S256 of the verifier")
	}

	parsed, err := url.Parse(NewOAuth().LoginURL("tdd_abc", "http://localhost:1/callback", "xyz", challenge))
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"client_id":             "tdd_abc",
		"state":                 "xyz",
		"scope":                 "data:read",
		"redirect_uri":          "http://localhost:1/callback",
		"code_challenge":        challenge,
		"code_challenge_method": "S256",
		"response_type":         "code",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestExchangeTurnsACodeIntoCredentials(t *testing.T) {
	oauth := fakeOAuth(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "the-code" || r.Form.Get("code_verifier") != "the-verifier" {
			t.Errorf("form = %v", r.Form)
		}
		w.Write([]byte(`{"access_token": "access-1", "refresh_token": "refresh-1", "expires_in": 3600}`))
	})

	creds, err := oauth.Exchange("tdd_abc", "http://localhost:1/callback", "the-code", "the-verifier")
	if err != nil {
		t.Fatal(err)
	}
	if creds.ClientID != "tdd_abc" || creds.AccessToken != "access-1" || creds.RefreshToken != "refresh-1" {
		t.Errorf("creds = %+v", creds)
	}
	if left := time.Until(creds.ExpiresAt); left < 59*time.Minute || left > 61*time.Minute {
		t.Errorf("expires in %v, want about an hour", left)
	}
}

func TestRejectedCodeIsReportedAsSuch(t *testing.T) {
	oauth := fakeOAuth(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	})

	if _, err := oauth.Exchange("tdd_abc", "http://localhost:1/callback", "bad", "v"); !errors.Is(err, ErrTokenRejected) {
		t.Fatalf("err = %v, want ErrTokenRejected", err)
	}
}

func TestTokenIsUsedUntilItExpiresThenRefreshedAndSaved(t *testing.T) {
	t.Setenv(tokenEnv, "")
	refreshes := 0
	oauth := fakeOAuth(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-1" || r.Form.Get("client_id") != "tdd_abc" {
			t.Errorf("form = %v", r.Form)
		}
		refreshes++
		w.Write([]byte(`{"access_token": "access-2", "refresh_token": "refresh-2", "expires_in": 3600}`))
	})
	path := filepath.Join(t.TempDir(), "todoist-auth.json")

	if _, err := oauth.Token(path); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("err = %v, want ErrNotConnected", err)
	}

	fresh := Credentials{ClientID: "tdd_abc", AccessToken: "access-1", RefreshToken: "refresh-1", ExpiresAt: time.Now().Add(time.Hour)}
	if err := SaveCredentials(path, fresh); err != nil {
		t.Fatal(err)
	}
	if token, err := oauth.Token(path); err != nil || token != "access-1" || refreshes != 0 {
		t.Fatalf("fresh: token = %q, err = %v, refreshes = %d", token, err, refreshes)
	}

	stale := fresh
	stale.ExpiresAt = time.Now().Add(-time.Minute)
	if err := SaveCredentials(path, stale); err != nil {
		t.Fatal(err)
	}
	if token, err := oauth.Token(path); err != nil || token != "access-2" || refreshes != 1 {
		t.Fatalf("stale: token = %q, err = %v, refreshes = %d", token, err, refreshes)
	}

	saved, err := LoadCredentials(path)
	if err != nil || saved.AccessToken != "access-2" || saved.RefreshToken != "refresh-2" || saved.ClientID != "tdd_abc" {
		t.Fatalf("saved = %+v, err = %v", saved, err)
	}
}

func TestPastedTokenNeverExpiresAndEnvironmentWins(t *testing.T) {
	t.Setenv(tokenEnv, "")
	oauth := fakeOAuth(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("a pasted token must not be refreshed")
	})
	path := filepath.Join(t.TempDir(), "todoist-auth.json")

	if err := SaveCredentials(path, Credentials{AccessToken: "personal"}); err != nil {
		t.Fatal(err)
	}
	if token, err := oauth.Token(path); err != nil || token != "personal" {
		t.Fatalf("token = %q, err = %v", token, err)
	}

	t.Setenv(tokenEnv, "from-env")
	if token, _ := oauth.Token(path); token != "from-env" {
		t.Fatalf("token = %q, want the environment's", token)
	}
}
