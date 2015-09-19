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

var certFile = flag.String("certFile", "cert.pem", "SSL certificate")
var keyFile = flag.String("keyFile", "key.pem", "SSL certificate")
var addr = flag.String("addr", ":443", "https list addr")
var thisUrl = flag.String("url", "", "public url e.g. https://foo (required)")
var uberClientId = flag.String("uberClientId", "", "Uber client_id (required)")
var uberClientSecret = flag.String("uberClientSecret", "", "Uber client_secret (required)")
var uberApiHost = flag.String("uberApi", "https://api.uber.com/v1", "Uber API URL")
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

	uberAuthorizeUrl := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=%s&state=%s", UberAuthHost, *uberClientId, sessionId)
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
	redirectUri := fmt.Sprintf("%s%s", *thisUrl, redirectUriPath)

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
	mondoWebhookUrl := fmt.Sprintf("%s%s", *thisUrl, mondoWebhookPath)
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
	loginSuccessTemplate.Execute(w, r)
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
		fmt.Printf("%s ignored transaction: %s", request.Data.Description)
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

	uberMapResponse, err := uberApiClient.GetMap(session.uberAccessToken, requestId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s get map error: %s", SetAuthCode, err.Error())
		return
	}

	// Todo: look up Uber product types
	feedItemTitle := fmt.Sprintf("%s UberX %s", uberReceiptResponse.TotalCharged, uberHistoryItem.StartCity.DisplayName)

	err = mondoApiClient.CreateFeedItem(
		session.mondoAccessToken,
		session.mondoAccountId,
		"image",
		"Uber Receipt",
		uberMapResponse.Href,
		feedItemTitle)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%s create feed item error: %s", SetAuthCode, err.Error())
		return
	}
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
	if *uberClientId == "" || *uberClientSecret == "" || *thisUrl == "" {
		flag.PrintDefaults()
		return
	}
	uberApiClient = &UberApiClient{clientSecret: *uberClientSecret, clientId: *uberClientId}
	mondoApiClient = &MondoApiClient{url: *mondoApiUrl}
	router.HandleFunc("/", indexGet).Methods("GET").Name(Index)
	router.HandleFunc("/login", loginPost).Methods("POST").Name(Login)
	router.HandleFunc("/uber/setauthcode", uberSetAuthCodeGet).Methods("GET").Name(SetAuthCode)
	router.HandleFunc("/mondo/webhook/{sessionId}", mondoWebhookPost).Methods("POST").Name(MondoWebhook)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./")))
	log.Printf("Listening on %s\n", *addr)
	if strings.Contains(*addr, "443") {
		log.Fatal(http.ListenAndServeTLS(*addr, *certFile, *keyFile, middleware(router)))
	} else {
		log.Fatal(http.ListenAndServe(*addr, middleware(router)))
	}
}
