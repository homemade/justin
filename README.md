# justin (JustGiving Interface) [![GoDoc](https://godoc.org/github.com/homemade/justin?status.svg)](https://godoc.org/github.com/homemade/justin)

`justin` is a Go library providing a higher-level wrapper around the [JustGiving API](https://api.justgiving.com/docs)

## Overview

`justin` provides some higher-level functionality to the [JustGiving API](https://api.justgiving.com/docs) and is intended to speed up development of server-to-server interactions such as writing a micro service to interact with JustGiving.

The [JustGiving API](https://api.justgiving.com/docs) has a simple authentication model, all that is required is an API key.
Some API methods require user account authentication which is facilitated through a Basic Authentication header, `justin` will not store any user credentials so these API methods will simply have user name and password as part of their method signature.

### Validation

`justin` tries not to stand in the way of what you might want to send to JustGiving via their API and does not perform any validation prior to sending a request. However `justin` does like to try and be helpful. If a request fails, validation will then be run to augment the standard error message. The validation methods are also available on both the `models` and `justin.Service` for you to use if you wish.

## Running the tests

Set a `JUSTIN_APIKEY` env. var. to the API key to use for testing

Set a `JUSTIN_USER` env. var. to the email and password of the user account to use for testing (in user:password format)

Set a `JUSTIN_CHARITY` env. var. to the charity id to use for testing

Set a `JUSTIN_EVENT` env. var. to the event id to use for testing

### Including account admin tests
`go test -v -acc` using the `-acc` flag will create an account for the user as set through the env. vars. and send an email reminder to the newly created account

e.g. to just create an account and check it worked run `go test -v -acc -run="TestAccountAvailabilityCheck"`

### Standard tests
`go test -v` expects the user account set through the env. vars. to already exist

## Getting Started

Creating the service:

```go

  // Generated API Key accessible through a JustGiving developer account
  apiKey := "abc12345"
  // Timeout for the API requests
  timeout := time.Duration(20) * time.Second
  // JustGiving Environment to use
  env := justin.Sandbox
  svc, err := justin.CreateWithAPIKey(justin.APIKeyContext{
    APIKey: apiKey, Env: env, Timeout: timeout,
  })
  if err != nil {
    // ...
  }
  // Optionally a logger can be specified to log raw http requests/responses
  //()
  var logger api.LoggerFunc
  logger = func(m string) (err error) {
    fmt.Printf("[%v] %s", time.Now(), m))
    return nil
  }
  svcWithLogger, err := justin.CreateWithAPIKey(justin.APIKeyContext{
    APIKey: apiKey, Env: env, Timeout: timeout, HTTPLogger: logger,
  })
  if err != nil {
    // ...
  }

  // Use svc / svcWithLogger ...
```

### AccountAvailabilityCheck

Check the availability of a JustGiving account by email address:

```go
  eml, err := mail.ParseAddress("rob@golang.org")
  if err != nil {
    // ...
  }
  avail, err = svc.AccountAvailabilityCheck(*eml)
```

### Validate

Validate a set of supplied user credentials against the JustGiving database:

```go
  eml, err := mail.ParseAddress("rob@golang.org")
  if err != nil {
    // ...
  }
  pwd := "goph3r"
  valid, err := svc.Validate(*eml, pwd)
```

### AccountRegistration

Create a new user account with JustGiving:

```go
eml, err := mail.ParseAddress("john@justgiving.com")
if err != nil {
  // ...
}
acc := models.Account{
  Title:        "Mr",
  FirstName:    "John",
  LastName:     "Smith",
  Email:        *eml,
  Password:     "S3cr3tP4ssw0rd",
  AddressLine1: "Second Floor, Blue Fin Building",
  AddressLine2: "110 Southwark Street",
  County:       "London",
  TownOrCity:   "London",
  Postcode:     "SE1 0TA",
  Country:      "United Kingdom",
}
err = svc.AccountRegistration(acc)
```

### RequestPasswordReminder

Sends a password reset email:

```go
eml, err := mail.ParseAddress("rob@golang.org")
if err != nil {
  // ...
}
err := svc.RequestPasswordReminder(*eml)
```

### FundraisingPageUrlCheck

Checks the availability of a JustGiving fundraising page

```go
pageShortName := "robspage"
avail, suggestions, err := svc.FundraisingPageUrlCheck(pageShortName)
// If the page is not available some alternative page short names
// will be returned as suggestions []string
```


### RegisterFundraisingPage

Registers a Fundraising Page on the JustGiving website
```go
  eml, err := mail.ParseAddress("john@justgiving.com")
  if err != nil {
    // ...
  }
  pwd := "S3cr3tP4ssw0rd"
  var imgs [2]models.Image
  url, err := url.Parse("http://images.justgiving.com/image/image1.jpg")
  if err != nil {
    // ...
  }
  imgs[0] = models.Image{Caption: "Image 1 Caption", URL: *url}
  url, err = url.Parse("http://images.justgiving.com/image/image2.png")
  if err != nil {
    // ...
  }
  imgs[1] = models.Image{Caption: "Image 2 Caption", URL: *url}
  var cuscodes [6]string
  cuscodes[0] = "CUSTOMCODE1"
  // Can have up to 6 custom codes...
  cuscodes[5] = "CUSTOMCODE6"
  pg := models.FundraisingPageForEvent{
    CharityID:       123,
    EventID:         456789,
    PageShortName:   "johnstestpage",
    PageTitle:       "Page Title",
    PageStory:       "Page Story",
    Images:          imgs[:],
    TargetAmount:    "100.00",
    CustomCodes:     cuscodes,
    CurrencyCode:    "GBP",
    CharityFunded:   false,
    JustGivingOptIn: false,
    CharityOptIn:    false,
  }
  pageURL, signOnURL, err := s.RegisterFundraisingPageForEvent(*eml, pwd, pg)
  // You can redirect your users to the returned signOnURL and they will be
  // automatically signed in to their newly created fundraising page.
  // The URL can only be used once, and must be used within 20 minutes of
  // the page being created.
```

## Roadmap
Add Team API methods
https://api.justgiving.com/docs/resources/v1/Team
