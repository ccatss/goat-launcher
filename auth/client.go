package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/auroradevllc/apiclient"
	"golang.org/x/oauth2"
)

type Session struct {
	SessionID string `json:"sessionId"`
}

type SessionRequest struct {
	Token string `json:"idToken"`
}

type Account struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	UserHash    string `json:"userHash"`
}

func NewClient() *Client {
	return &Client{}
}

type Client struct {
}

type UserInfo struct {
	Sub              string   `json:"sub"`
	Nickname         string   `json:"nickname"`
	Issuer           string   `json:"iss"`
	Audience         []string `json:"aud"`
	AuthMethods      []string `json:"amr"`
	AuthTime         int64    `json:"auth_time"`
	IssuedAt         int64    `json:"iat"`
	ResourceAuthTime int64    `json:"rat"`
}

func (c *Client) UserInfo(token *oauth2.Token) (*UserInfo, error) {
	req, err := apiclient.NewRequest(userInfoUrl,
		apiclient.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken)))

	if err != nil {
		return nil, err
	}

	res, err := req.Send()

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}

	var info UserInfo

	if err := res.Unmarshal(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (c *Client) CreateSession(token string) (*Session, error) {
	req, err := apiclient.NewRequest(sessionUrl,
		apiclient.WithMethod(http.MethodPost),
		apiclient.WithHeader("Accept", "application/json"),
		apiclient.WithJSON(SessionRequest{Token: token}))

	if err != nil {
		return nil, err
	}

	res, err := req.Send()

	if err != nil {
		return nil, err
	}

	b, err := res.Bytes()

	if err != nil {
		return nil, err
	}

	fmt.Println(string(b))

	var session Session

	if err := json.Unmarshal(b, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (c *Client) Accounts(s *Session) ([]Account, error) {
	req, err := apiclient.NewRequest(accountsUrl,
		apiclient.WithHeader("Accept", "application/json"),
		apiclient.WithHeader("Authorization",
			fmt.Sprintf("Bearer %s", s.SessionID)))

	if err != nil {
		return nil, err
	}

	res, err := req.Send()

	if err != nil {
		return nil, err
	}

	var accounts []Account

	if err := res.Unmarshal(&accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}
