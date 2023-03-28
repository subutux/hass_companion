package auth

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/k0kubun/pp/v3"
)

type Credentials struct {
	Server       string
	ClientId     string
	Token        string
	accessToken  string
	RefreshToken string
	Expires      time.Time
	TokenType    string
}

type AuthorizationResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func NewCredentials(server, clientId, accessToken, refreshToken string) Credentials {
	return Credentials{
		Server:       server,
		ClientId:     clientId,
		accessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

func (c *Credentials) Authorize() error {
	if !c.shouldAuthorize() {
		return nil
	}
	api := resty.New()
	endpoint, _ := url.Parse(c.Server)
	endpoint.Path = "/auth/token"
	formData := map[string]string{
		"grant_type": "authorization_code",
		"code":       c.Token,
		"client_id":  c.ClientId,
	}
	pp.Println(formData)
	response, err := api.R().SetFormData(formData).SetResult(&AuthorizationResponse{}).
		Post(endpoint.String())

	if err != nil {
		return err
	}

	if !response.IsSuccess() {
		return fmt.Errorf("Failed to fetch token from authorization_code: %s", response.String())
	}

	return c.setTokensFromResponse(response.Result().(*AuthorizationResponse))

}

func (c *Credentials) setTokensFromResponse(authorization *AuthorizationResponse) error {
	log.Printf("Setting accessToken = %s", authorization.AccessToken)
	c.accessToken = authorization.AccessToken
	if authorization.RefreshToken != "" {
		log.Printf("Setting RefreshToken = %s", authorization.RefreshToken)
		c.RefreshToken = authorization.RefreshToken
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

func (c *Credentials) shouldAuthorize() bool {
	return c.RefreshToken == ""
}

func (c *Credentials) refresh() error {
	if !c.shouldRefresh() {
		return nil
	}
	log.Print("refreshing token")
	endpoint, _ := url.Parse(c.Server)
	endpoint.Path = "/auth/token"
	api := resty.New()
	response, err := api.R().SetFormData(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": c.RefreshToken,
		"client_id":     c.ClientId,
	}).SetResult(&AuthorizationResponse{}).
		Post(endpoint.String())

	log.Printf("Status: %v data: %s", response.Status(), response.String())

	if err != nil {

		log.Printf("error requesting token: %s", err)
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
