package api

import (
	"text/template"
)

const accountRegistrationTmpl = `{
    "acceptTermsAndConditions": true,
    "address": {
        "country": "{{.Country}}",
        "countyOrState": "{{.County}}",
        "line1": "{{.AddressLine1}}",
        "line2": "{{.AddressLine2}}",
        "postcodeOrZipcode": "{{.Postcode}}",
        "townOrCity": "{{.TownOrCity}}"
    },
    "causeId": null,
    "email": "{{.PlainEmail}}",
    "firstName": "{{.FirstName}}",
    "lastName": "{{.LastName}}",
    "password": "{{.Password}}",
    "reference": null,
    "title": "{{.Title}}"
}`

const registerFundraisingPageForEventTmpl = `{
  "charityId": {{.CharityID}},
  "eventId": {{.EventID}},
  "pageShortName": "{{.PageShortName}}",
  "pageTitle": "{{.PageTitle}}",
  "targetAmount": "{{.TargetAmount}}",
  "justGivingOptIn": {{.JustGivingOptIn}},
  "charityOptIn": {{.CharityOptIn}},
  "charityFunded": {{.CharityFunded}},
  "pageStory": "{{.PageStory}}",
  "customCodes": {
    "customCode1": "{{index .CustomCodes 0}}",
    "customCode2": "{{index .CustomCodes 1}}",
    "customCode3": "{{index .CustomCodes 2}}",
    "customCode4": "{{index .CustomCodes 3}}",
    "customCode5": "{{index .CustomCodes 4}}",
    "customCode6": "{{index .CustomCodes 5}}"
  },{{ if gt (len .Images) 0 }}"images": [
    {{range $i, $v := .Images}}{{if ne $i 0}},{{end}}{"caption": "{{$v.Caption}}","url": "{{$v.URL}}","isDefault": "{{eq $i 0}}"}{{ end }}
    ],{{ end }}
  "currency": "{{.CurrencyCode}}"{{ if gt .TeamID 0 }},
  "teamId": {{.TeamID}}{{ end }}
}`

const validateTmpl = `{
    "email": "{{.Email}}",
    "password": "{{.Password}}"
}`

func init() {
	// Cache request templates
	RequestTemplates = make(map[string]RequestTemplate)
	// Validate
	validate := RequestTemplate{}
	validate.t, validate.err = template.New("validateTmpl").Parse(validateTmpl)
	RequestTemplates["Validate"] = validate
	// AccountRegistration
	accountRegistration := RequestTemplate{}
	accountRegistration.t, accountRegistration.err = template.New("accountRegistrationTmpl").Parse(accountRegistrationTmpl)
	RequestTemplates["AccountRegistration"] = accountRegistration
	// RegisterFundraisingPageForEvent
	registerFundraisingPageForEvent := RequestTemplate{}
	registerFundraisingPageForEvent.t, registerFundraisingPageForEvent.err = template.New("registerFundraisingPageForEventTmpl").Parse(registerFundraisingPageForEventTmpl)
	RequestTemplates["RegisterFundraisingPageForEvent"] = registerFundraisingPageForEvent
}
