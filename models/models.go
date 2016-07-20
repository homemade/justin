// Package models provides types representing JustGiving objects
package models

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ParseDate attempts to convert the date string returned by JustGiving to a Time
//
// The raw date is returned in the follwing format `/Date(1474675200000+0000)/``
//
// Parsing is carried out as follows:
// 	- Remove the leading `/Date(` string
//  - Extract the remaining integer value prefixing the `+`
//  - Divide this number by 1000
//  - Use https://golang.org/pkg/time/#Time.Unix to return a Time from the result
func ParseDate(date string) (time.Time, error) {
	var result time.Time
	s := date
	if s == "" {
		return result, errors.New("no value set for EventDate")
	}
	s = strings.Replace(s, "/Date(", "", -1)
	i := strings.Index(s, "+")
	if i < 4 {
		return result, errors.New("invalid format for EventDate")
	}
	s = s[0:i]
	ui, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return result, err
	}
	if ui < 1000 {
		return result, errors.New("invalid format for EventDate")
	}
	return time.Unix(ui/1000, 0), nil
}
