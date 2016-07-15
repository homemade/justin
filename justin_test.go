package justin

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/homemade/justin/api"
	"github.com/homemade/justin/models"
)

const (
	APIKeyEnvVar  = "JUSTIN_APIKEY"
	UserEnvVar    = "JUSTIN_USER"
	CharityEnvVar = "JUSTIN_CHARITY"
	EventEnvVar   = "JUSTIN_EVENT"
)

var (
	runAccountAdminTests bool
	createdAccount       bool
)

func TestMain(m *testing.M) {
	flag.BoolVar(&runAccountAdminTests, "acc", false, "run account admin tests to create user account and send password reminder email")
	flag.Parse()
	os.Exit(m.Run())
}

func createService(t *testing.T, env Env) *Service {
	// Get API key from env var
	apiKey := ev(APIKeyEnvVar, t)
	// Set a timeout for our API requests
	tim := time.Duration(20) * time.Second
	// Log http requests/responses to std out
	logger := api.StructuredLogger(os.Stdout)
	// Create the service
	svc, err := CreateWithAPIKey(APIKeyContext{
		APIKey: apiKey, Env: env, Timeout: tim, HTTPLogger: logger,
	})
	if err != nil {
		t.Fatal(err)
		return nil
	}
	// Optionally test account registration / creation and send password reminder email
	if runAccountAdminTests && !createdAccount {
		createdAccount = testAccountRegistration(t, svc)
		if !createdAccount {
			t.Fatal("failed to create account when running tests with -acc flag")
			return nil
		}
		// send password reminder
		sent := testRequestPasswordReminder(t, svc)
		if !sent {
			t.Fatal("failed to send password reminder email when running tests with -acc flag")
			return nil
		}
	}
	svc.TraceOrigin("unit testing")
	return svc
}

func ev(name string, t *testing.T) string {
	result := os.Getenv(name)
	if result == "" {
		t.Fatalf("missing env var %s", name)
	}
	return result
}

func e(t *testing.T, err error) bool {
	if err != nil {
		t.Error(err)
		return true
	}
	return false
}

func getUserCreds(t *testing.T) (username string, pwd string, err error) {
	usrEV := ev(UserEnvVar, t)
	parts := strings.Split(usrEV, ":")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return username, pwd, fmt.Errorf("invalid or missing env var %s", UserEnvVar)
	}
	return parts[0], parts[1], nil
}

func getPage(pageURL url.URL) (string, error) {
	res, err := http.Get(pageURL.String())
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func inPage(html string, search ...string) bool {
	for _, s := range search {
		if !strings.Contains(html, s) {
			return false
		}
	}
	return true

}

func testAccountAvailabilityCheck(t *testing.T, s *Service) {
	// Available
	eml, err := mail.ParseAddress("averyunusualemailthatnoonewouldevercreateanaccountwith@justgiving.com")
	if e(t, err) {
		return
	}
	avail, err := s.AccountAvailabilityCheck(*eml)
	if e(t, err) {
		return
	}
	if !avail {
		t.Errorf("expected AccountAvailabilityCheck to return true but returned %t", avail)
	}
	// UnAvailable
	userEmail, _, err := getUserCreds(t)
	if e(t, err) {
		return
	}
	eml, err = mail.ParseAddress(userEmail)
	if e(t, err) {
		return
	}
	avail, err = s.AccountAvailabilityCheck(*eml)
	if e(t, err) {
		return
	}
	if avail {
		t.Errorf("expected AccountAvailabilityCheck to return false but returned %t", avail)
		return
	}
}

func TestAccountAvailabilityCheck(t *testing.T) {
	// Sandbox test
	s := createService(t, Sandbox)
	testAccountAvailabilityCheck(t, s)
}

func testValidate(t *testing.T, s *Service) {
	// Valid
	userEmail, pwd, err := getUserCreds(t)
	if e(t, err) {
		return
	}
	eml, err := mail.ParseAddress(userEmail)
	if e(t, err) {
		return
	}
	valid, err := s.Validate(*eml, pwd)
	if e(t, err) {
		return
	}
	if !valid {
		t.Errorf("expected Validate to return true but returned %t", valid)
	}
	// Invalid
	eml, err = mail.ParseAddress("invaliduser@justgiving.com")
	if e(t, err) {
		return
	}
	valid, err = s.Validate(*eml, "invalidpassword")
	if e(t, err) {
		return
	}
	if valid {
		t.Errorf("expected Validate to return false but returned %t", valid)
	}
}

func TestValidate(t *testing.T) {
	// Sandbox test
	s := createService(t, Sandbox)
	testValidate(t, s)
}

func testAccountRegistration(t *testing.T, s *Service) bool {
	userEmail, pwd, err := getUserCreds(t)
	if e(t, err) {
		return false
	}
	eml, err := mail.ParseAddress(userEmail)
	if e(t, err) {
		return false
	}
	// Invalid Country - expect to fail
	invalidAccount := models.Account{
		Title:        "Mr",
		FirstName:    "John",
		LastName:     "Smith",
		Email:        *eml,
		Password:     pwd,
		AddressLine1: "Second Floor, Blue Fin Building",
		AddressLine2: "110 Southwark Street",
		County:       "London",
		TownOrCity:   "London",
		Postcode:     "SE1 0TA",
		Country:      "Nowhere",
	}
	err = s.AccountRegistration(invalidAccount)
	if err == nil {
		t.Error("expected AccountRegistration to return error due to invalid Country")
		return false
	}
	if err.Error() != "invalid response 400 Bad request, result of running validation on request payload was: invalid Country" {
		t.Errorf("expected AccountRegistration to return error due to invalid Country but recieved error %v", err)
		return false
	}
	// Valid (depending on the email env var)
	validAccount := invalidAccount
	validAccount.Country = "United Kingdom"
	err = s.AccountRegistration(validAccount)
	if e(t, err) {
		return false
	}

	return true
}

func testRequestPasswordReminder(t *testing.T, s *Service) bool {
	userEmail, _, err := getUserCreds(t)
	if e(t, err) {
		return false
	}
	eml, err := mail.ParseAddress(userEmail)
	if e(t, err) {
		return false
	}
	err = s.RequestPasswordReminder(*eml)
	if e(t, err) {
		return false
	}
	return true
}

func testFundraisingPageAPI(t *testing.T, s *Service) {
	userEmail, pwd, err := getUserCreds(t)
	if e(t, err) {
		return
	}
	eml, err := mail.ParseAddress(userEmail)
	if e(t, err) {
		return
	}
	charityID, err := strconv.Atoi(ev(CharityEnvVar, t))
	if e(t, err) {
		return
	}
	eventID, err := strconv.Atoi(ev(EventEnvVar, t))
	if e(t, err) {
		return
	}
	// Create a page
	pgsn := "testpage" + time.Now().Format("20060102150405")
	var imgs [2]models.Image
	url, err := url.Parse("http://images.justgiving.com/image/dad9226d-bfb5-4ba0-af1f-c64f5afa9ef9.jpg")
	if e(t, err) {
		return
	}
	imgs[0] = models.Image{Caption: "Image 1 Caption", URL: *url}
	url, err = url.Parse("http://images.justgiving.com/image/e7048a7f-567c-4d1c-8b66-693af8be696f.png")
	if e(t, err) {
		return
	}
	imgs[1] = models.Image{Caption: "Image 2 Caption", URL: *url}
	var cuscodes [6]string
	cuscodes[0] = "CUSTOMCODE1"
	cuscodes[5] = "CUSTOMCODE6"
	pg := models.FundraisingPageForEvent{
		CharityID:       uint(charityID),
		EventID:         uint(eventID),
		PageShortName:   pgsn,
		PageTitle:       "Page Title For " + pgsn,
		PageStory:       "Page Story For " + pgsn,
		Images:          imgs[:],
		TargetAmount:    "100.00",
		CustomCodes:     cuscodes,
		CurrencyCode:    "GBP",
		CharityFunded:   false,
		JustGivingOptIn: false,
		CharityOptIn:    false,
	}

	pageURL, signOnURL, err := s.RegisterFundraisingPageForEvent(*eml, pwd, pg)
	if e(t, err) {
		return
	}

	// Check pageURL looks OK
	html, err := getPage(*pageURL)
	if e(t, err) {
		return
	}
	if !inPage(html, pg.PageTitle, pg.PageStory, pg.TargetAmount) {
		f := pgsn + ".fail.html"
		t.Errorf("the created fundraising page (checked with returned page url) does not look as expected, see %s", f)
		err = ioutil.WriteFile(f, []byte(html), 0644)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	// Check signOnURL looks OK
	html, err = getPage(*signOnURL)
	if !inPage(html, pg.PageTitle, pg.PageStory, pg.TargetAmount) {
		f := pgsn + ".fail.html"
		t.Errorf("the created fundraising page (checked with returned signon url) does not look as expected, see %s", f)
		err = ioutil.WriteFile(f, []byte(html), 0644)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	// Check page exists as far as JustGiving is concerned
	avail, suggs, err := s.FundraisingPageURLCheck(pgsn)
	if e(t, err) {
		return
	}
	if avail {
		t.Errorf("expected FundraisingPageUrlCheck to return false but returned %t", avail)
		return
	}

	// Check we got some suggestions
	if len(suggs) < 1 {
		t.Error("expected FundraisingPageUrlCheck to return some suggestions")
		return
	}

	// Try and create it again - should fail
	_, _, err = s.RegisterFundraisingPageForEvent(*eml, pwd, pg)
	if err == nil {
		t.Error("expected RegisterFundraisingPageForEvent to return error as page already registered")
		return
	}

	// Check a non existent page doesn't exist
	avail, _, err = s.FundraisingPageURLCheck("hopefullythispagenamewillneverexist")
	if !avail {
		t.Errorf("expected FundraisingPageUrlCheck to return true but returned %t", avail)
		return
	}

	// Check we can retrieve all the pages
	pages, err := s.FundraisingPagesForEvent(uint(eventID))
	if err != nil {
		t.Fatal(err)
	}

	//Check fundraising results are as expected
	if len(pages) > 0 {
		pgfr, err := s.FundraisingPageResults(pages[len(pages)-1]) // our most recent one should be last
		if err != nil {
			t.Fatal(err)
			return
		}
		if pg.TargetAmount != pgfr.Target {
			t.Errorf("the fundraising page results are not as expected, see %#v", pgfr)
			return
		}
		t.Logf("FundraisingResults %#v", pgfr)
	}

	// Check we can retrieve the pages based on the charity and user
	_, err = s.FundraisingPagesForCharityAndUser(uint(charityID), *eml)
	if err != nil {
		t.Fatal(err)
	}

}

func TestFundraisingPageAPI(t *testing.T) {
	// Sandbox test
	s := createService(t, Sandbox)
	testFundraisingPageAPI(t, s)

}
