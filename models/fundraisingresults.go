package models

import "time"

// FundraisingResults contains the current fundraising results as provided by JustGiving
type FundraisingResults struct {
	Target                        string `json:"fundraisingTarget"`
	TotalRaisedPercentageOfTarget string `json:"totalRaisedPercentageOfFundraisingTarget"`
	TotalRaisedOffline            string `json:"totalRaisedOffline"`
	TotalRaisedOnline             string `json:"totalRaisedOnline"`
	TotalRaisedSMS                string `json:"totalRaisedSms"`
	TotalEstimatedGiftAid         string `json:"totalEstimatedGiftAid"`
	EventDate                     string `json:"eventDate"`
	PageCancelled                 bool
}

// ParseEventDate attempts to convert the EventDate returned by JustGiving to a Time
func (r FundraisingResults) ParseEventDate() (time.Time, error) {
	return ParseDate(r.EventDate)
}
