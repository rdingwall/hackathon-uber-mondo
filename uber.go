package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	url          string
}

type UberTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint32 `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type UberHistoryResponse struct {
	Offset  int64             `json:"offset"`
	Limit   int64             `json:"limit"`
	Count   int64             `json:"count"`
	History []UberHistoryItem `json:"history"`
}

type UberHistoryItem struct {
	Status      string          `json:"status"`
	Distance    float64         `json:"distance"`
	RequestTime int64           `json:"request_time"`
	StartTime   int64           `json:"start_time"`
	StartCity   UberHistoryCity `json:"start_city"`
	EndTime     int64           `json:"end_time"`
	RequestId   string          `json:"request_id"`
	ProductId   string          `json:"product_id"`
}

type UberHistoryCity struct {
	Latitude    float64 `json:"latitude"`
	DisplayName string  `json:"display_name"`
	Longitude   float64 `json:"longitude"`
}

type UberReceiptResponse struct {
	RequestId     string `json:"request_id"`
	TotalCharged  string `json:"total_charged"`
	Distance      string `json:"distance"`
	DistanceLabel string `json:"miles"`
}

type UberRequestResponse struct {
	Status   string              `json:"status"`
	Location UberRequestLocation `json:"location"`
}

type UberRequestLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (c *UberApiClient) GetOAuthToken(authorizationCode, redirectUri string) (*UberTokenResponse, error) {
	uberTokenUrl := fmt.Sprintf("%s/oauth/token", UberAuthHost)
	formValues := url.Values{
		"client_secret": {c.clientSecret},
		"client_id":     {c.clientId},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectUri},
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

		body, err := ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(string(body))
	}

	uberTokenResponse := &UberTokenResponse{}
	err = json.NewDecoder(httpResponse.Body).Decode(uberTokenResponse)
	return uberTokenResponse, err
}

func (c *UberApiClient) GetHistory(accessToken string) (*UberHistoryResponse, error) {
	// Hide cancelled & giving transactions for demo...
	uberHistoryUrl := fmt.Sprintf("%s/v1.2/history?offset=3", c.url)
	request, err := http.NewRequest("GET", uberHistoryUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add(Authorization, Bearer+accessToken)

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

	uberHistoryResponse := &UberHistoryResponse{}
	err = json.NewDecoder(response.Body).Decode(uberHistoryResponse)
	if err != nil {
		return nil, err
	}

	return uberHistoryResponse, nil
}

func (c *UberApiClient) GetReceipt(accessToken, requestId string) (*UberReceiptResponse, error) {
	uberHistoryUrl := fmt.Sprintf("%s/v1/requests/%s/receipt", c.url, requestId)
	request, err := http.NewRequest("GET", uberHistoryUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add(Authorization, Bearer+accessToken)

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

	uberReceiptResponse := &UberReceiptResponse{}
	err = json.NewDecoder(response.Body).Decode(uberReceiptResponse)
	if err != nil {
		return nil, err
	}

	return uberReceiptResponse, nil
}

func (c *UberApiClient) GetRequest(accessToken, requestId string) (*UberRequestResponse, error) {
	uberHistoryUrl := fmt.Sprintf("%s/v1/requests/%s", c.url, requestId)
	request, err := http.NewRequest("GET", uberHistoryUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add(Authorization, Bearer+accessToken)

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

	uberRequestResponse := &UberRequestResponse{}
	err = json.NewDecoder(response.Body).Decode(uberRequestResponse)
	if err != nil {
		return nil, err
	}

	return uberRequestResponse, nil
}
