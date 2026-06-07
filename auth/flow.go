package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	ErrStateMismatch = errors.New("state mismatch")
	scopes           = []string{"openid", "offline", "gamesso.token.create", "user.profile.read"}
)

const (
	authUrl          = "https://account.jagex.com/oauth2/auth"
	tokenUrl         = "https://account.jagex.com/oauth2/token"
	userInfoUrl      = "https://account.jagex.com/userinfo"
	sessionUrl       = "https://auth.jagex.com/game-session/v1/sessions"
	accountsUrl      = "https://auth.jagex.com/game-session/v1/accounts"
	jagexLauncherID  = "com_jagex_auth_desktop_launcher"
	consentClientID  = "1fddee4e-b100-4f4e-b2b0-097f9088f9d2"
	launcherRedirect = "https://secure.runescape.com/m=weblogin/launcher-redirect"
)

type Launcher func(url string) error

type Flow struct {
	conf          *oauth2.Config
	verifier      string
	expectedState string
	launcher      Launcher
	token         *oauth2.Token
	complete      func(string, *oauth2.Token)
}

func NewAuthFlow(launcher Launcher, complete func(string, *oauth2.Token)) (*Flow, error) {
	conf := &oauth2.Config{
		ClientID:    jagexLauncherID,
		RedirectURL: launcherRedirect,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authUrl,
			TokenURL: tokenUrl,
		},
		Scopes: scopes,
	}

	verifier := oauth2.GenerateVerifier()

	f := &Flow{
		conf:     conf,
		verifier: verifier,
		launcher: launcher,
		complete: complete,
	}

	if err := f.regenerateState(); err != nil {
		return nil, err
	}

	return f, nil
}

func (a *Flow) regenerateState() error {
	state, err := uuid.NewV7()

	if err != nil {
		return err
	}

	a.expectedState = state.String()
	return nil
}

func (a *Flow) Start() error {
	authUrl := a.conf.AuthCodeURL(a.expectedState, oauth2.S256ChallengeOption(a.verifier))

	return a.launcher(authUrl)
}

func (a *Flow) Exchange(code, state string) error {
	if a.expectedState != state {
		return ErrStateMismatch
	}

	token, err := a.conf.Exchange(context.Background(), code, oauth2.VerifierOption(a.verifier))

	if err != nil {
		return err
	}

	a.token = token

	return a.handleConsent(token)
}

func (a *Flow) handleConsent(token *oauth2.Token) error {
	if err := a.regenerateState(); err != nil {
		return err
	}

	// Replace config with our consent config
	a.conf = &oauth2.Config{
		ClientID:    consentClientID,
		RedirectURL: "http://localhost",
		Endpoint: oauth2.Endpoint{
			AuthURL:  authUrl,
			TokenURL: tokenUrl,
		},
		Scopes: []string{"openid", "offline"},
	}

	nonce, err := uuid.NewV7()

	if err != nil {
		return err
	}

	consentUrl := a.conf.AuthCodeURL(a.expectedState,
		oauth2.VerifierOption(a.verifier),
		oauth2.SetAuthURLParam("response_type", "id_token code"),
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("nonce", nonce.String()),
		oauth2.SetAuthURLParam("id_token_hint", token.Extra("id_token").(string)))

	return a.launcher(consentUrl)
}

func (a *Flow) Consent(idToken string) {
	a.complete(idToken, a.token)
}
