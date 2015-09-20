package main

import (
	_ "crypto/sha512"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nu7hatch/gouuid"
	"html/template"
	"log"
	"net/http"
	"strings"
)

const (
	// Route names
	Index        = "/"
	Login        = "/login"
	Logout       = "/logout"
	SetAuthCode  = "/uber/setauthcode"
	ReceiptReady = "/uber/webooks/requests.receipt_ready"
	UberWebhook  = "/uber/webhook"
	MondoWebhook = "/mondo/webhook"
)

type session struct {
	sessionId        string
	mondoAccessToken string
	mondoAccountId   string
	mondoWebhookId   string
	uberAccessToken  string
}

var googleMapsApiKey = flag.String("gMapsApiKey", "", "Google Maps API key (required)")
var certFile = flag.String("certFile", "cert.pem", "SSL certificate")
var keyFile = flag.String("keyFile", "key.pem", "SSL certificate")
var httpsAddr = flag.String("https", ":443", "HTTPS address to bind on")
var httpAddr = flag.String("http", ":80", "HTTP address to bind on")
var httpsUrl = flag.String("httpsUrl", "", "public HTTPS URL for Uber redirect e.g. https://foo (required)")
var httpUrl = flag.String("httpUrl", "", "public HTTP URL for Mondo webhook e.g. http://foo (required)")
var uberClientId = flag.String("uberClientId", "", "Uber client_id (required)")
var uberClientSecret = flag.String("uberClientSecret", "", "Uber client_secret (required)")
var uberApiHost = flag.String("uberApi", "https://api.uber.com", "Uber API URL (no trailing slash)")
var mondoApiUrl = flag.String("mondoApi", "https://api.getmondo.co.uk", "Mondo API URL")

var indexTemplate = template.Must(template.ParseFiles("index.html"))
var pleaseWaitTemplate = template.Must(template.ParseFiles("pleasewait.html"))
var loginSuccessTemplate = template.Must(template.ParseFiles("loginsuccess.html"))

var sessions = make(map[string]*session)
var router = mux.NewRouter()
var uberApiClient *UberApiClient
var mondoApiClient *MondoApiClient

func indexGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTemplate.Execute(w, r.Host)
}

func loginPost(w http.ResponseWriter, r *http.Request) {
	mondoAccessToken := r.FormValue("mondo-access-token")
	mondoAccountId := r.FormValue("mondo-account-id")

	if mondoAccessToken == "" || mondoAccountId == "" {
		http.Error(w, "required: mondo-access-token, mondo-account-id", http.StatusBadRequest)
		log.Printf("%s required: mondo-access-token, mondo-account-id", Login)
		return
	}

	// Register session
	uuid, err := uuid.NewV4()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s generate session id error: %s", Login, err.Error())
		return
	}

	sessionId := uuid.String()
	session := &session{
		sessionId:        sessionId,
		mondoAccessToken: mondoAccessToken,
		mondoAccountId:   mondoAccountId}

	sessions[sessionId] = session

	uberAuthorizeUrl := fmt.Sprintf("%s/oauth/authorize?response_type=code&scope=history request request_receipt&client_id=%s&state=%s", UberAuthHost, *uberClientId, sessionId)
	log.Printf("redirecting to %s", uberAuthorizeUrl)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ UberAuthorizeUrl string }{UberAuthorizeUrl: uberAuthorizeUrl}
	pleaseWaitTemplate.Execute(w, data)
}

func uberSetAuthCodeGet(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query()["state"][0]
	session, exists := sessions[sessionId]
	if !exists {
		http.Error(w, fmt.Sprintf("No such session %s", sessionId), http.StatusNotFound)
		return
	}

	redirectUriPath, err := router.Get(SetAuthCode).URLPath()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s error: %s", SetAuthCode, err.Error())
		return
	}
	redirectUri := fmt.Sprintf("%s%s", *httpsUrl, redirectUriPath)

	uberAuthorizationCode := r.URL.Query()["code"][0]
	uberTokenResponse, err := uberApiClient.GetOAuthToken(uberAuthorizationCode, redirectUri)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s uber oauth token error: %s", SetAuthCode, err.Error())
		return
	}

	session.uberAccessToken = uberTokenResponse.AccessToken
	log.Printf("%s assigned session id=%s Uber access_token=%s\n", SetAuthCode, sessionId, uberTokenResponse.AccessToken)

	// Register Mondo webhook
	mondoWebhookPath, err := router.Get(MondoWebhook).URLPath("sessionId", sessionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s error: %s", SetAuthCode, err.Error())
		return
	}
	mondoWebhookUrl := fmt.Sprintf("%s%s", *httpUrl, mondoWebhookPath)
	log.Printf("%s registering mondo webhook url=%s", SetAuthCode, mondoWebhookUrl)
	mondoWebhookResponse, err := mondoApiClient.RegisterWebHook(session.mondoAccessToken, session.mondoAccountId, mondoWebhookUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s register mondo webhook error: %s", SetAuthCode, err.Error())
		return
	}

	session.mondoWebhookId = mondoWebhookResponse.Webhook.Id
	log.Printf("%s successfully registered mondo webhook id=%s", SetAuthCode, mondoWebhookResponse.Webhook.Id)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ SessionId string }{SessionId: sessionId}
	loginSuccessTemplate.Execute(w, data)
}

func mondoWebhookPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request = &WebhookRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("%s json parse error: %s", MondoWebhook, err.Error())
		return
	}

	if !strings.Contains(strings.ToUpper(request.Data.Description), "UBER") {
		fmt.Printf("%s ignored transaction: %s", MondoWebhook, request.Data.Description)
		return
	}

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]
	session, exists := sessions[sessionId]
	if !exists {
		http.Error(w, fmt.Sprintf("No such session %s", sessionId), http.StatusNotFound)
		return
	}

	uberHistoryResponse, err := uberApiClient.GetHistory(session.uberAccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s get history error: %s", SetAuthCode, err.Error())
		return
	}

	uberHistoryItem := uberHistoryResponse.History[0]
	requestId := uberHistoryItem.RequestId
	uberReceiptResponse, err := uberApiClient.GetReceipt(session.uberAccessToken, requestId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s get receipt error: %s", SetAuthCode, err.Error())
		return
	}

	uberRequestResponse, err := uberApiClient.GetRequest(session.uberAccessToken, requestId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s get request error: %s", SetAuthCode, err.Error())
		return
	}

	start := coordinate{
		Latitude:  uberHistoryItem.StartCity.Latitude,
		Longitude: uberHistoryItem.StartCity.Longitude,
	}

	end := coordinate{
		Latitude:  uberRequestResponse.Location.Latitude,
		Longitude: uberRequestResponse.Location.Longitude,
	}

	feedItemImageUrl := googleMapsUrl(start, end, *googleMapsApiKey)

	feedItemTitle := fmt.Sprintf("%s %s", uberReceiptResponse.TotalCharged, uberHistoryItem.StartCity.DisplayName)

	err = mondoApiClient.CreateFeedItem(
		session.mondoAccessToken,
		session.mondoAccountId,
		"image",
		fmt.Sprintf("Uber Receipt %s", randomCarEmoji()),
		feedItemImageUrl,
		feedItemTitle)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s create feed item error: %s", SetAuthCode, err.Error())
		return
	}
}

func logoutPost(w http.ResponseWriter, r *http.Request) {
	sessionId := r.FormValue("sessionId")
	session, exists := sessions[sessionId]
	if !exists {
		http.Error(w, fmt.Sprintf("No such session %s", sessionId), http.StatusNotFound)
		return
	}

	mondoApiClient.UnregisterWebHook(session.mondoAccessToken, session.mondoWebhookId)
}

func middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s\n", r.Method, r.URL)
		h.ServeHTTP(w, r)
	})
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()
	if *uberClientId == "" || *uberClientSecret == "" || *httpsUrl == "" || *httpUrl == "" || *googleMapsApiKey == "" {
		flag.PrintDefaults()
		return
	}
	uberApiClient = &UberApiClient{
		url:          *uberApiHost,
		clientSecret: *uberClientSecret,
		clientId:     *uberClientId,
	}

	mondoApiClient = &MondoApiClient{url: *mondoApiUrl}

	router.HandleFunc("/", indexGet).Methods("GET").Name(Index)
	router.HandleFunc("/login", loginPost).Methods("POST").Name(Login)
	router.HandleFunc("/logout", logoutPost).Methods("POST").Name(Logout)
	router.HandleFunc("/uber/setauthcode", uberSetAuthCodeGet).Methods("GET").Name(SetAuthCode)
	router.HandleFunc("/mondo/webhook/{sessionId}", mondoWebhookPost).Methods("POST").Name(MondoWebhook)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./")))

	go func() {
		log.Printf("Listening on %s\n", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, middleware(router)))
	}()

	log.Printf("Listening on %s\n", *httpsAddr)
	if strings.Contains(*httpsAddr, "443") {
		log.Fatal(http.ListenAndServeTLS(*httpsAddr, *certFile, *keyFile, middleware(router)))
	} else {
		log.Fatal(http.ListenAndServe(*httpsAddr, middleware(router)))
	}
}
