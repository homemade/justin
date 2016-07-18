// Package justin is a client library and higher-level wrapper around the JustGiving API (https://api.justgiving.com/docs)
package justin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"time"

	"github.com/homemade/justin/api"
	"github.com/homemade/justin/models"
)

// Env represents a JustGiving Environment
type Env int

const (
	// Version is the current release
	Version = "1.1.0"

	// UserAgent is set to identify justin requests
	UserAgent = "justin " + Version

	// ContentType is that used in the JustGiving API requests/responses
	ContentType = "application/json"

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
	origin string
}

// APIKeyContext contains settings for creating a justin Service with an API Key.
//
// HTTPLogger is an optional implementation of the Logger interface, if not provided no logging will be carried out
type APIKeyContext struct {
	APIKey     string
	Env        Env
	Timeout    time.Duration
	HTTPLogger api.Logger
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

// TraceOrigin will augment any logging with the specified origin
func (svc *Service) TraceOrigin(origin string) {
	svc.origin = origin
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

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return false, err
	}

	res, _, err := api.Do(svc.client, svc.origin, "AccountAvailabilityCheck", req, "", svc.HTTPLogger)
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

	sBody, body, err := api.BuildBody("Validate", data, ContentType)
	if err != nil {
		return false, err
	}

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), body)
	if err != nil {
		return false, err
	}

	res, resBody, err := api.Do(svc.client, svc.origin, "Validate", req, sBody, svc.HTTPLogger)
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

	sBody, body, err := api.BuildBody("AccountRegistration", account, ContentType)
	if err != nil {
		return err
	}
	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), body)
	if err != nil {
		return err
	}

	res, _, err := api.Do(svc.client, svc.origin, "AccountRegistration", req, sBody, svc.HTTPLogger)
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

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return false, err
	}

	res, resBody, err := api.Do(svc.client, svc.origin, "IsValidCountry", req, "", svc.HTTPLogger)
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

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return err
	}

	res, _, err := api.Do(svc.client, svc.origin, "RequestPasswordReminder", req, "", svc.HTTPLogger)
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

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return false, suggs, err
	}

	res, _, err := api.Do(svc.client, svc.origin, "FundraisingPageURLCheck", req, "", svc.HTTPLogger)
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
	path.WriteString(url.QueryEscape(pageShortName))
	req, err = api.BuildRequest(UserAgent, ContentType, "GET", path.String(), nil)
	res, resBody, err := api.Do(svc.client, svc.origin, "FundraisingPageURLCheck", req, "", svc.HTTPLogger)
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

	sBody, body, err := api.BuildBody("RegisterFundraisingPageForEvent", page, ContentType)
	if err != nil {
		return nil, nil, err
	}
	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), body)
	if err != nil {
		return nil, nil, err
	}

	// This request requires authentication
	// mail.Address stores email in the format <rob@golang.org>, we don't want the `<` `>`
	em := account.String()
	em = em[1 : len(em)-1]
	req.SetBasicAuth(em, password)

	res, resBody, err := api.Do(svc.client, svc.origin, "RegisterFundraisingPageForEvent", req, sBody, svc.HTTPLogger)
	if err != nil {
		return nil, nil, err
	}

	//201 Created
	if res.StatusCode != 201 {
		// run request validation on failure
		var info string
		var valid bool
		valid, err = page.HasValidCurrencyCode(svc)
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
	if err = json.Unmarshal([]byte(resBody), &result); err != nil {
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

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return false, err
	}

	res, resBody, err := api.Do(svc.client, svc.origin, "IsValidCurrencyCode", req, "", svc.HTTPLogger)
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

// FundraisingPageRef represents a valid reference to a JustGiving fundraising page
type FundraisingPageRef struct {
	charityID uint

	eventID uint

	id uint

	shortName string
}

func (r *FundraisingPageRef) CharityID() uint {
	return r.charityID
}
func (r *FundraisingPageRef) EventID() uint {
	return r.eventID
}
func (r *FundraisingPageRef) ID() uint {
	return r.id
}
func (r *FundraisingPageRef) ShortName() string {
	return r.shortName
}

// FundraisingPageResults returns the current fundraising results for the specified JustGiving page
func (svc *Service) FundraisingPageResults(page *FundraisingPageRef) (models.FundraisingResults, error) {

	var result models.FundraisingResults
	method := "GET"
	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/fundraising/pages/")
	path.WriteString(page.shortName)

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return result, err
	}

	res, resBody, err := api.Do(svc.client, svc.origin, "FundraisingPageResults", req, "", svc.HTTPLogger)
	if err != nil {
		return result, err
	}

	if res.StatusCode == 410 {
		result.Cancelled = true
		return result, nil
	}

	if res.StatusCode != 200 {
		return result, fmt.Errorf("invalid response %s", res.Status)
	}

	result = models.FundraisingResults{}
	if err = json.Unmarshal([]byte(resBody), &result); err != nil {
		return result, fmt.Errorf("invalid response %v", err)
	}

	return result, nil

}

// FundraisingPagesForCharityAndUser returns the charity's fundraising pages registered with the specified JustGiving user account
func (svc *Service) FundraisingPagesForCharityAndUser(charityID uint, account mail.Address) ([]*FundraisingPageRef, error) {

	var results []*FundraisingPageRef

	// mail.Address stores email in the format <rob@golang.org>, this simply removes the `<` `>`
	em := account.String()
	if em != "" {
		em = em[1 : len(em)-1]
	}

	method := "GET"
	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/account/")
	path.WriteString(em)
	path.WriteString("/pages/?charityId=")
	path.WriteString(strconv.FormatUint(uint64(charityID), 10))

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String(), nil)
	if err != nil {
		return nil, err
	}
	res, resBody, err := api.Do(svc.client, svc.origin, "FundraisingPagesForCharityAndUser", req, "", svc.HTTPLogger)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("invalid response %s", res.Status)
	}

	var result = []struct {
		EventID       uint   `json:"eventId"`
		PageID        uint   `json:"pageId"`
		PageShortName string `json:"pageShortName"`
	}{}

	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return nil, fmt.Errorf("invalid response %v", err)
	}

	for _, p := range result {
		if p.PageID > 0 {
			results = append(results, &FundraisingPageRef{
				charityID: charityID,
				eventID:   p.EventID,
				id:        p.PageID,
				shortName: p.PageShortName,
			})

		}
	}

	return results, nil
}

// FundraisingPagesForEvent returns the fundraising pages registered for the specified event
func (svc *Service) FundraisingPagesForEvent(eventID uint) ([]*FundraisingPageRef, error) {

	results, totalPagination, totalFundraisingPages, err := paginatedFundraisingPagesForEvent(svc, eventID, 0)
	if err != nil {
		return nil, err
	}
	if totalPagination > 1 {
		for i := 2; i <= int(totalPagination); i++ {
			var nextResults []*FundraisingPageRef
			nextResults, totalPagination, totalFundraisingPages, err = paginatedFundraisingPagesForEvent(svc, eventID, uint(i))
			if err != nil {
				return nil, err
			}
			for _, nr := range nextResults {
				results = append(results, nr)
			}
		}
	}

	if int(totalFundraisingPages) != len(results) {
		return results, fmt.Errorf("inconsistent read, expected %d results but have %d", int(totalFundraisingPages), len(results))
	}

	return results, nil
}
