package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

// http://developer.okta.com/docs/api/rest/sessions.html
type sessionRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionResponseSuccess struct {
	Id          string
	UserID      string
	MfaActive   bool
	CookieToken string
}

type configuration struct {
	OktaAPIKey       string
	OktaHost         string
	DefaultTargetURL string
	GuestUserName    string
	GuestPassword    string
}

var appConfig configuration

func main() {

	loadConfig()

	http.HandleFunc("/guest", guestLogin)
	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/", rootHandler)

	var portNumber string = os.Getenv("PORT")

	if portNumber == "" {
		portNumber = "9000"
	}

	fmt.Println("listening on port:" + portNumber)
	err := http.ListenAndServe(":"+portNumber, nil)
	if err != nil {
		panic(err)
	}
}

func rootHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "This page intentionally left blank")
}

func guestLogin(res http.ResponseWriter, req *http.Request) {

	Apiurl := appConfig.OktaHost + "/api/v1/sessions?additionalFields=cookieToken"

	// fmt.Printf("Starting Request from " + req.RemoteAddr + " - URL:" + req.RequestURI + "\n")

	request := sessionRequest{Username: appConfig.GuestUserName, Password: appConfig.GuestPassword}
	// fmt.Printf("jsonRqst: %s \n", request)

	jsonRqst, err := json.Marshal(request)

	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}

	// fmt.Printf("jsonRqst: %s \n", jsonRqst)

	r, _ := http.NewRequest("POST", Apiurl, bytes.NewReader(jsonRqst))
	r.Header.Add("Authorization", "SSWS "+appConfig.OktaAPIKey)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Accept", "application/json")

	//fmt.Printf("r: %s \n", r)

	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}

	// fmt.Printf("%s", resp)

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		OktaBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			fmt.Printf("Error: %s", err)
			return
		}

		var sessionResp sessionResponseSuccess
		err = json.Unmarshal(OktaBody, &sessionResp)

		// Need to REDIRECT to this URL template:
		//    https://your-subdomain.okta.com/login/sessionCookieRedirect?token={cookieToken}&redirectUrl={redirectUrl}

		var redirectURLValue string
		redirectURLValue = req.FormValue("redirectUrl")
		if redirectURLValue == "" {

			redirectURLValue = url.QueryEscape(appConfig.DefaultTargetURL)
		}

		//fmt.Println(redirectURLValue)

		OKTASessURL := appConfig.OktaHost + "/login/sessionCookieRedirect?token=" + sessionResp.CookieToken + "&redirectUrl=" + redirectURLValue

		fmt.Println(OKTASessURL)
		// Do NOT cache anything incase user comes back an hour or so later, thier guest session would have expired

		res.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
		res.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
		res.Header().Set("Expires", "0")                                         // Proxies

		http.Redirect(res, req, OKTASessURL, 301)

	} else {
		// TODO-add some code here.
		fmt.Println("Something is wrong")
	}
}

func aboutHandler(res http.ResponseWriter, req *http.Request) {
	viewData := make(map[string]string)

	var tmpl = template.Must(template.ParseFiles("templates/about.html"))
	if req.Method == "POST" {

		var selfReferencingURL string
		selfReferencingURL = "http://" + req.Host + "/guest?redirectUrl=" + url.QueryEscape(req.FormValue("redirectUrl"))
		// fmt.Println(selfReferencingURL)

		// viewData["encodedURL"] = url.QueryEscape(req.FormValue("redirectUrl"))
		viewData["encodedURL"] = selfReferencingURL
	}

	err := tmpl.Execute(res, viewData)

	if err != nil {
		fmt.Println("executing template:", err)
	}

}

func loadConfig() {
	// Load config file
	jsonConfigFile, err := os.Open("config/config.json")
	if err != nil {
		panic("Configuration file 'config/config.json' could not be found")
	}
	fmt.Printf("----\nConfiguration File Found. Hooray!! \n")

	defer jsonConfigFile.Close()

	decoder := json.NewDecoder(jsonConfigFile)
	appConfig = configuration{}

	err = decoder.Decode(&appConfig)

	if err != nil {
		fmt.Println("error:", err)
		panic("Configuration file could not be found.")
	}
	fmt.Printf("OKTA HOST: %s \n", appConfig.OktaHost)
	fmt.Printf("Guest User Name from Config File: %s \n----\n", appConfig.GuestUserName)

}
