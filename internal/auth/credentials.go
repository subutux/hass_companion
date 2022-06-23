package auth

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type Credentials struct {
	Server       string
	ClientId     string
	Token        string
	accessToken  string
	refreshToken string
	Expires      time.Time
	TokenType    string
}

type AuthorizationResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func (c *Credentials) Authorize() error {
	api := resty.New()
	endpoint, _ := url.Parse(c.Server)
	endpoint.Path = "/auth/token"
	response, err := api.R().SetFormData(map[string]string{
		"grant_type": "authorization_code",
		"code":       c.Token,
		"client_id":  c.ClientId,
	}).SetResult(&AuthorizationResponse{}).
		Post(endpoint.String())

	if err != nil {
		return err
	}

	if !response.IsSuccess() {
		return fmt.Errorf("Failed to fetch token from authorization_code: %s", response.String())
	}

	authorization := response.Result().(*AuthorizationResponse)
	log.Printf("Setting accessToken = %s", authorization.AccessToken)
	c.accessToken = authorization.AccessToken
	if authorization.RefreshToken != "" {
		log.Printf("Setting RefreshToken = %s", authorization.RefreshToken)
		c.refreshToken = authorization.RefreshToken
	}
	duration, err := time.ParseDuration(strconv.Itoa(authorization.ExpiresIn) + "s")

	now := time.Now()
	c.Expires = now.Add(duration)

	if err != nil {
		return err
	}
	return nil
}

func (c *Credentials) setTokensFromResponse(authorization *AuthorizationResponse) error {
	log.Printf("Setting accessToken = %s", authorization.AccessToken)
	c.accessToken = authorization.AccessToken
	if authorization.RefreshToken != "" {
		log.Printf("Setting RefreshToken = %s", authorization.RefreshToken)
		c.refreshToken = authorization.RefreshToken
	}
	duration, err := time.ParseDuration(strconv.Itoa(authorization.ExpiresIn) + "s")

	now := time.Now()
	c.Expires = now.Add(duration)

	if err != nil {
		return err
	}
	return nil
}

func (c *Credentials) shouldRefresh() bool {
	return time.Now().After(c.Expires)
}

func (c *Credentials) refresh() error {
	if !c.shouldRefresh() {
		return nil
	}

	endpoint, _ := url.Parse(c.Server)
	endpoint.Path = "/auth/token"
	api := resty.New()
	response, err := api.R().SetFormData(map[string]string{
		"grant_type":    "rerfresh_token",
		"refresh_token": c.refreshToken,
		"client_id":     c.ClientId,
	}).SetResult(&AuthorizationResponse{}).
		Post(endpoint.String())

	if err != nil {
		return err
	}

	if !response.IsSuccess() {
		return errors.New("Failed to refresh token")
	}

	return c.setTokensFromResponse(response.Result().(*AuthorizationResponse))
}

func (c Credentials) AccessToken() string {
	c.refresh()
	return c.accessToken
}
