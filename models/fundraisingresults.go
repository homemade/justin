package models

// FundraisingResults contains the current fundraising results as provided by JustGiving
type FundraisingResults struct {
	Target                        string `json:"fundraisingTarget"`
	TotalRaisedPercentageOfTarget string `json:"totalRaisedPercentageOfFundraisingTarget"`
	TotalRaisedOffline            string `json:"totalRaisedOffline"`
	TotalRaisedOnline             string `json:"totalRaisedOnline"`
	TotalRaisedSMS                string `json:"totalRaisedSms"`
	TotalEstimatedGiftAid         string `json:"totalEstimatedGiftAid"`
}

// page id, date, FundraisingResults_id
// is 3 cols year, month, day better than date?
// run every hour updating above
