package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

const (
	UberAuthHost = "https://login.uber.com"
)

type UberApiClient struct {
	clientSecret string
	clientId     string
}

type UberTokenResponse struct {
	AccessToken  string `json:"acces_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint32 `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type UberWebookRequest struct {
	EventId      string          `json:"event_id"`
	EventTime    int64           `json:"event_time"`
	EventType    string          `json:"event_type"`
	Meta         uberWebhookMeta `json:"meta"`
	ResourceHref string          `json:"resource_href"`
}

type uberWebhookMeta struct {
	ResourceId   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Status       string `json:"status"`
}

func (c *UberApiClient) OAuthAuthorize(sessionId string, redirectUrl string) error {
	uberAuthorizeUrl := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s", UberAuthHost, *uberClientId, redirectUrl)
	log.Printf("%s requesting %s\n", Login, uberAuthorizeUrl)
	httpResponse, err := http.Get(uberAuthorizeUrl)
	if err != nil {
		log.Printf("%s uber authorize error: %s", Login, err.Error())
		return err
	}

	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != 200 {
		if err != nil {
			log.Printf("%s uber authorize response error: %s", Login, err.Error())
			return err
		}

		return errors.New(fmt.Sprintf("%s uber authorize response contained error, status=%s", Login, httpResponse.Status))
	}

	return nil
}

func (c *UberApiClient) GetOAuthToken(authorizationCode string) (*UberTokenResponse, error) {
	uberTokenUrl := fmt.Sprintf("%s/oauth/token", UberAuthHost)
	formValues := url.Values{
		"client_secret": {c.clientSecret},
		"client_id":     {c.clientId},
		"grant_type":    {"authorization_code"},
		"code":          {authorizationCode},
	}

	log.Printf("%s requesting %s\n", SetAuthCode, uberTokenUrl)
	httpResponse, err := http.PostForm(uberTokenUrl, formValues)

	if err != nil {
		log.Printf("/login uber authorize error: %s", err.Error())
		return nil, err
	}

	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != 200 {
		if err != nil {
			log.Printf("%s Uber OAuth token response error: %s", Login, err.Error())
			return nil, err
		}

		return nil, errors.New(fmt.Sprintf("%s Uber OAuth token response contained error, status=%s", Login, httpResponse.Status))
	}

	uberTokenResponse := &UberTokenResponse{}
	err = json.NewDecoder(httpResponse.Body).Decode(uberTokenResponse)
	return uberTokenResponse, err
}

func (c *UberApiClient) GetReceipt() {

}

func (c *UberApiClient) GetMap() {

}
