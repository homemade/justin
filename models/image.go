package models

import (
	"net/url"
)

type Image struct {
	Caption string
	URL     url.URL
}
