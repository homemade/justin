package models

import (
	"strconv"
)

// FundraisingPageForEventValidationService defines the validation methods requiring calls to the JustGiving API.
//
// For an implementation see justin.Service
type FundraisingPageForEventValidationService interface {
	IsValidCurrencyCode(currencyCode string) (bool, error)
}

// FundraisingPageForEvent represents a JustGiving fundraising page for a pre-defined JustGiving event
type FundraisingPageForEvent struct {
	CharityID uint

	EventID uint

	PageShortName string

	PageTitle string

	PageStory string

	Images []Image

	CustomCodes [6]string

	// TargetAmount for this fundraising effort expressed as a valid currency amount e.g. "999.99" or "9999"
	TargetAmount string

	// CurrencyCode
	CurrencyCode string

	CharityFunded bool

	JustGivingOptIn bool

	CharityOptIn bool

	TeamID uint
}

// HasValidCurrencyCode checks the CurrencyCode is in the published JustGiving currency code list
func (fp FundraisingPageForEvent) HasValidCurrencyCode(vs FundraisingPageForEventValidationService) (bool, error) {
	return vs.IsValidCurrencyCode(fp.CurrencyCode)
}

// HasValidTargetAmount performs basic validation on the TargetAmount
func (fp FundraisingPageForEvent) HasValidTargetAmount() bool {
	if fp.TargetAmount == "" {
		return true
	}
	_, err := strconv.ParseFloat(fp.TargetAmount, 64)
	if err != nil {
		return false
	}
	return true
}
