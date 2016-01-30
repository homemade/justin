// Package justin is a client library and higher-level wrapper around the JustGiving API (https://api.justgiving.com/docs)
package justin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"time"

	"github.com/homemade/justin/api"
	"github.com/homemade/justin/models"
)

// Env represents a JustGiving Environment
type Env int

const (
	// Version is the current release
	Version = "1.0.0"

	// UserAgent is set to identify justin requests
	UserAgent = "justin " + Version

	// TODO cache from Sandbox using https://github.com/homemade/ersatz?
	Local Env = iota

	// Sandbox represents the JustGiving sandbox environment (https://api.sandbox.justgiving.com)
	Sandbox

	// Live represents the JustGiving production environment (https://api.justgiving.com)
	Live

	sandboxBasePath = "https://api.sandbox.justgiving.com"
	liveBasePath    = "https://api.justgiving.com"
)

// Service provides a client library and higher-level wrapper around the JustGiving API (https://api.justgiving.com/docs)
type Service struct {

	// APIKeyContext used to create this service
	APIKeyContext

	// BasePath for the JustGiving API endpoint - based on the Env
	BasePath string

	client *http.Client
}

// Logger provides a simple interface for logging API calls made using justin
type Logger interface {
	Log(m string) (err error)
}

// LoggerFunc provides a simple type for single function implementations of the Logger interface
type LoggerFunc func(m string) (err error)

// Log defines the single method Logger interface
func (f LoggerFunc) Log(m string) (err error) {
	return f(m)
}

// APIKeyContext contains settings for creating a justin Service with an API Key.
//
// HTTPLogger is an optional implementation of the Logger interface, if not provided no logging will be carried out
type APIKeyContext struct {
	APIKey     string
	Env        Env
	Timeout    time.Duration
	HTTPLogger Logger
}

// CreateWithAPIKey instantiates the Service using an APIKey for authentication
func CreateWithAPIKey(api APIKeyContext) (svc *Service, err error) {
	// Create service
	svc = &Service{
		APIKeyContext: api,
		client:        &http.Client{Timeout: api.Timeout},
	}
	switch api.Env {
	case Sandbox:
		svc.BasePath = sandboxBasePath
	case Live:
		svc.BasePath = liveBasePath
	}

	// Check it works
	eml, err := mail.ParseAddress("webmaster@justgiving.com")
	if err != nil {
		return nil, fmt.Errorf("error creating test email to validate api key %v", err)
	}
	_, err = svc.AccountAvailabilityCheck(*eml)
	if err != nil {
		return nil, fmt.Errorf("error validating api key %v", err)
	}
	return svc, nil
}

// AccountAvailabilityCheck checks the availability of a JustGiving account by email address
func (svc *Service) AccountAvailabilityCheck(account mail.Address) (avail bool, err error) {

	method := "HEAD"

	// mail.Address stores email in the format <rob@golang.org>, we don't want the `<` `>`
	em := account.String()
	em = em[1 : len(em)-1]

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/account/")
	path.WriteString(em)

	req, err := api.BuildRequest(UserAgent, method, path.String(), nil)
	if err != nil {
		return false, err
	}

	res, _, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return false, err
	}

	// 404 is success (available), which is a bit dangerous, so we first make sure we have the correct JustGiving response header
	if res.Header.Get("X-Justgiving-Operation") != "AccountApi:AccountAvailabilityCheck" {
		return false, fmt.Errorf("invalid response, expected X-Justgiving-Operation response header to be AccountApi:AccountAvailabilityCheck but recieved %s", res.Header.Get("X-Justgiving-Operation"))
	}
	if res.StatusCode == 404 {
		return true, nil
	}
	if res.StatusCode != 200 {
		return false, fmt.Errorf("invalid response %s", res.Status)
	}
	// 200 - account exists
	return false, nil

}

// Validate a set of supplied user credentials against the JustGiving database
func (svc *Service) Validate(account mail.Address, password string) (valid bool, err error) {

	method := "POST"

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/account/validate")

	// mail.Address stores email in the format <rob@golang.org>, we don't want the `<` `>`
	em := account.String()
	em = em[1 : len(em)-1]

	data := struct {
		Email    string
		Password string
	}{em, password}
	body, err := api.BuildBody("Validate", data)
	if err != nil {
		return false, err
	}

	req, err := api.BuildRequest(UserAgent, method, path.String(), body)
	if err != nil {
		return false, err
	}

	res, resBody, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return false, err
	}

	if res.StatusCode != 200 {
		return false, fmt.Errorf("invalid response %s", res.Status)
	}
	var result = struct {
		IsValid bool `json:"isValid"`
	}{}
	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return false, fmt.Errorf("invalid response %v", err)
	}
	return result.IsValid, nil

}

// AccountRegistration registers a new user account with JustGiving
func (svc *Service) AccountRegistration(account models.Account) (err error) {

	method := "PUT"

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/account/")

	body, err := api.BuildBody("AccountRegistration", account)
	if err != nil {
		return err
	}
	req, err := api.BuildRequest(UserAgent, method, path.String(), body)
	if err != nil {
		return err
	}

	res, _, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		// run request validation on failure
		info := "no errors found"
		valid, err := account.HasValidCountry(svc)
		if err != nil {
			info = fmt.Sprintf("errors running validation %v", err)
		} else {
			if !valid {
				info = "invalid Country"
			}
		}
		return fmt.Errorf("invalid response %s, result of running validation on request payload was: %s", res.Status, info)
	}
	return nil

}

// IsValidCountry checks the Country used by models.Account is in the published JustGiving countries list
func (svc *Service) IsValidCountry(name string) (bool, error) {

	method := "GET"

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/countries")

	req, err := api.BuildRequest(UserAgent, method, path.String(), nil)
	if err != nil {
		return false, err
	}

	res, resBody, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return false, err
	}

	if res.StatusCode != 200 {
		return false, fmt.Errorf("invalid response %s", res.Status)
	}

	var result = []struct {
		Name string `json:"name"`
	}{}

	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return false, fmt.Errorf("invalid response %v", err)
	}

	for _, c := range result {
		if name == c.Name {
			return true, nil
		}
	}
	return false, nil

}

// RequestPasswordReminder requests JustGiving to send a password reset email
func (svc *Service) RequestPasswordReminder(account mail.Address) error {

	method := "GET"

	// mail.Address stores email in the format <rob@golang.org>, we don't want the `<` `>`
	em := account.String()
	em = em[1 : len(em)-1]

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/account/")
	path.WriteString(em)
	path.WriteString("/requestpasswordreminder")

	req, err := api.BuildRequest(UserAgent, method, path.String(), nil)
	if err != nil {
		return err
	}

	res, _, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("invalid response %s", res.Status)
	}
	return nil

}

// FundraisingPageURLCheck checks the availability of a JustGiving fundraising page
func (svc *Service) FundraisingPageURLCheck(pageShortName string) (avail bool, suggestions []string, err error) {

	// if page is not available we return some suggestions
	var suggs []string

	method := "HEAD"

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/fundraising/pages/")
	path.WriteString(pageShortName)

	req, err := api.BuildRequest(UserAgent, method, path.String(), nil)
	if err != nil {
		return false, suggs, err
	}

	res, _, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return false, suggs, err
	}

	// 404 is success, which is a bit dangerous, so we first make sure we have the correct JustGiving response header
	if res.Header.Get("X-Justgiving-Operation") != "FundraisingApi:FundraisingPageUrlCheck" {
		return false, suggs, fmt.Errorf("invalid response, expected X-Justgiving-Operation response header to be FundraisingApi:FundraisingPageUrlCheck but recieved %s", res.Header.Get("X-Justgiving-Operation"))
	}
	if res.StatusCode == 404 {
		return true, suggs, nil
	}
	if res.StatusCode != 200 {
		return false, suggs, fmt.Errorf("invalid response %s", res.Status)
	}
	// 200 - Page short name already registered
	// Return a list of suggestions
	path = bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/fundraising/pages/suggest?preferredName=")
	path.WriteString(pageShortName)
	req, err = api.BuildRequest(UserAgent, "GET", path.String(), nil)
	res, resBody, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return false, suggs, err
	}
	var result = struct {
		Names []string
	}{}
	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return false, suggs, fmt.Errorf("invalid response %v", err)
	}
	return false, result.Names, nil

}

// RegisterFundraisingPageForEvent registers a fundraising page on the JustGiving website
func (svc *Service) RegisterFundraisingPageForEvent(account mail.Address, password string, page models.FundraisingPageForEvent) (pageURL *url.URL, signOnURL *url.URL, err error) {

	method := "PUT"

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/fundraising/pages")

	body, err := api.BuildBody("RegisterFundraisingPageForEvent", page)
	if err != nil {
		return nil, nil, err
	}
	req, err := api.BuildRequest(UserAgent, method, path.String(), body)
	if err != nil {
		return nil, nil, err
	}

	// This request requires authentication
	// mail.Address stores email in the format <rob@golang.org>, we don't want the `<` `>`
	em := account.String()
	em = em[1 : len(em)-1]
	req.SetBasicAuth(em, password)

	res, resBody, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return nil, nil, err
	}

	//201 Created
	if res.StatusCode != 201 {
		// run request validation on failure
		var info string
		valid, err := page.HasValidCurrencyCode(svc)
		if err != nil {
			info = fmt.Sprintf("errors running CurrencyCode validation %v; ", err)
		} else {
			if !valid {
				info = "invalid CurrencyCode; "
			}
		}
		valid = page.HasValidTargetAmount()
		if !valid {
			info += "invalid TargetAmount"
		}
		if info == "" {
			info = "no errors found"
		}
		return nil, nil, fmt.Errorf("invalid response %s, result of running validation on request payload was: %s", res.Status, info)
	}

	// Read page URL and signon URL from response
	var result = struct {
		SignOnURL string `json:"signOnUrl"`
		Page      struct {
			URL string `json:"uri"`
		} `json:"next"`
	}{}
	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return nil, nil, fmt.Errorf("invalid response %v", err)
	}
	pageURL, err = url.Parse(result.Page.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid response %v", err)
	}
	signOnURL, err = url.Parse(result.SignOnURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid response %v", err)
	}

	return pageURL, signOnURL, nil

}

// IsValidCurrencyCode checks the CurrencyCode used by models.FundraisingPageForEvent is in the published JustGiving currency code list
func (svc *Service) IsValidCurrencyCode(code string) (bool, error) {
	method := "GET"

	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/fundraising/currencies")

	req, err := api.BuildRequest(UserAgent, method, path.String(), nil)
	if err != nil {
		return false, err
	}

	res, resBody, err := api.Do(svc.client, req, svc.HTTPLogger)
	if err != nil {
		return false, err
	}

	if res.StatusCode != 200 {
		return false, fmt.Errorf("invalid response %s", res.Status)
	}

	var result = []struct {
		Code string `json:"currencyCode"`
	}{}

	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return false, fmt.Errorf("invalid response %v", err)
	}

	for _, c := range result {
		if code == c.Code {
			return true, nil
		}
	}
	return false, nil

}
