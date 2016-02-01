package models

import (
	"net/url"
)

// Image represents a JustGiving fundraising page image
type Image struct {
	Caption string
	URL     url.URL
}
