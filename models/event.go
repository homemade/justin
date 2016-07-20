package models

import "time"

// Event represents a JustGiving fundraising event
type Event struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CompletionDate string `json:"completionDate"`
	ExpiryDate     string `json:"expiryDate"`
	StartDate      string `json:"startDate"`
	Type           string `json:"eventType"`
	Location       string `json:"location"`
}

// ParseCompletionDate attempts to convert the EventDate returned by JustGiving to a Time
func (e Event) ParseCompletionDate() (time.Time, error) {
	return ParseDate(e.CompletionDate)
}

// ParseExpiryDate attempts to convert the EventDate returned by JustGiving to a Time
func (e Event) ParseExpiryDate() (time.Time, error) {
	return ParseDate(e.ExpiryDate)
}

// ParseStartDate attempts to convert the EventDate returned by JustGiving to a Time
func (e Event) ParseStartDate() (time.Time, error) {
	return ParseDate(e.StartDate)
}
