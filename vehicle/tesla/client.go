package tesla

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/uhthomas/tesla"
	"golang.org/x/oauth2"
)

// Identity is the tesla authentication client
type Identity struct {
	Config   *oauth2.Config
	auth     *tesla.Auth
	verifier string
}

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// https://www.oauth.com/oauth2-servers/pkce/
func pkce() (verifier, challenge string, err error) {
	var p [87]byte
	if _, err := io.ReadFull(rand.Reader, p[:]); err != nil {
		return "", "", fmt.Errorf("rand read full: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(p[:])
	b := sha256.Sum256([]byte(challenge))
	challenge = base64.RawURLEncoding.EncodeToString(b[:])
	return verifier, challenge, nil
}

// NewIdentity creates a tesla authentication client
func NewIdentity(client *http.Client) (*Identity, error) {
	config := &oauth2.Config{
		ClientID:     "ownerapi",
		ClientSecret: "",
		RedirectURL:  "https://auth.tesla.com/void/callback",
		Scopes:       []string{"openid email offline_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.tesla.com/oauth2/v3/authorize",
			TokenURL: "https://auth.tesla.com/oauth2/v3/token",
		},
	}

	verifier, challenge, err := pkce()
	if err != nil {
		return nil, fmt.Errorf("pkce: %w", err)
	}

	auth := &tesla.Auth{
		Client: client,
		AuthURL: config.AuthCodeURL(state(), oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("code_challenge", challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		),
	}

	c := &Identity{
		Config:   config,
		auth:     auth,
		verifier: verifier,
	}
	c.DeviceHandler(c.mfaUnsupported)

	return c, nil
}

// Login executes the MFA or non-MFA login
func (c *Identity) Login(username, password string) (*oauth2.Token, error) {
	ctx := context.Background()
	code, err := c.auth.Do(ctx, username, password)
	if err != nil {
		return nil, err
	}

	token, err := c.Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", c.verifier),
	)

	return token, err
}

// DeviceHandler sets an alternative authentication device handler
func (c *Identity) DeviceHandler(handler func(context.Context, []tesla.Device) (tesla.Device, string, error)) {
	c.auth.SelectDevice = handler
}

func (c *Identity) mfaUnsupported(_ context.Context, _ []tesla.Device) (tesla.Device, string, error) {
	return tesla.Device{}, "", errors.New("multi factor authentication is not supported")
}
