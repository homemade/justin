package models

import (
	"net/mail"
)

type AccountValidationService interface {
	IsValidCountry(name string) (bool, error)
}

type Account struct {
	Title        string
	FirstName    string
	LastName     string
	Email        mail.Address
	Password     string
	AddressLine1 string
	AddressLine2 string
	County       string
	TownOrCity   string
	Postcode     string
	Country      string
}

// PlainEmail returns a plainer email address
// (mail.Address stores email in the format <rob@golang.org>, this simply removes the `<` `>`
func (acc Account) PlainEmail() string {
	em := acc.Email.String()
	if em == "" {
		return ""
	}
	return em[1 : len(em)-1]
}

func (acc Account) HasValidCountry(vs AccountValidationService) (bool, error) {
	return vs.IsValidCountry(acc.Country)
}
