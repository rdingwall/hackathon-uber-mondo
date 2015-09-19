package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	Authorization   = "Authorization"
	Bearer          = "Bearer "
	ContentType     = "Content-Type"
	ApplicationJson = "application/json"
)

type MondoApiClient struct {
	clientId     string
	clientSecret string
	url          string
}

type RegisterWebhookRequest struct {
	AccountId string `json:"account_id"`
	Url       string `json:"url"`
}

type Webhook struct {
	AccountId string `json:"account_id"`
	Id        string `json:"id"`
	Url       string `json:"url"`
}

type RegisterWebhookResponse struct {
	Webhook Webhook `json:"webhook"`
}

type WebhookRequest struct {
	Type string      `json:"type"`
	Data WebhookData `json:"data"`
}

type WebhookData struct {
	Amount      string `json:"amount"`
	Created     string `json:"created"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	Id          string `json:"id"`
}

var httpClient = &http.Client{}

func (c *MondoApiClient) RegisterWebHook(accessToken string, accountId string, url string) (*RegisterWebhookResponse, error) {
	webhookRequest := &RegisterWebhookRequest{AccountId: accountId, Url: url}
	buffer := &bytes.Buffer{}
	err := json.NewEncoder(buffer).Encode(webhookRequest)
	if err != nil {
		return nil, err
	}

	webhooksUrl := fmt.Sprintf("%s/webhooks", c.url)
	request, err := http.NewRequest("POST", webhooksUrl, buffer)
	if err != nil {
		return nil, err
	}

	request.Header.Add(Authorization, Bearer+accessToken)
	request.Header.Add(ContentType, ApplicationJson)

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(body))
	}

	webhookResponse := &RegisterWebhookResponse{}
	err = json.NewDecoder(response.Body).Decode(webhookResponse)
	if err != nil {
		return nil, err
	}

	return webhookResponse, nil
}
