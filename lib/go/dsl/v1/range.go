// SPDX-FileCopyrightText: © 2022 The mistral authors <github.com/worldiety/mistral.git/lib/go/dsl/AUTHORS>
// SPDX-License-Identifier: BSD-2-Clause

package miel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const dateTimeFormat = "2006-01-02 15:04:05"

// parserState represents a state in our finite state machine to parse the format as defined by Range.
type parserState int

const (
	psStart parserState = iota
	psMin
	psMax
	psTimeZone
)

// Range is a string representation of a range. ( or ] can be used to indicate inclusive and exclusive intervals.
// Format specification:
//  <[|(> <min>, <max> <]|)> @ <IANA time zone name>
// Examples:
//  [2038-01-19 03:14:07,2038-01-19 03:14:07]@Europe/Berlin
//  (2038-01-19 03:14:07,2038-01-19 03:14:07)@Europe/Berlin
type Range string

// UnmarshalJSON validates the range during unmarshalling.
func (r *Range) UnmarshalJSON(bytes []byte) error {
	s := string(bytes)
	s, err := strconv.Unquote(s)
	if err != nil {
		return fmt.Errorf("cannot unquote range: %w", err)
	}

	if _, _, err := Range(s).Interval(); err != nil {
		return fmt.Errorf("invalid range '%s': %w", s, err)
	}

	*r = Range(s)

	return nil
}

// MustInterval returns the inclusive Interval representation of this Range. See also Interval.
func (r Range) MustInterval() Interval {
	min, max, err := r.Interval()
	if err != nil {
		panic(err)
	}

	return Interval{
		Min: min,
		Max: max,
	}
}

// Interval parses and returns the min and max unix timestamps, which have always 'inclusive' semantics.
// Min and max are represented as a unix timestamp in seconds.
func (r Range) Interval() (min, max int64, err error) {
	str := strings.TrimSpace(string(r))
	state := psStart
	minIsInclusive := false
	maxIsInclusive := false

	minString := ""
	maxString := ""
	tzString := ""

	offset := 0

parserLoop:
	for i, r := range str {
		switch r {
		case '[':
			if err := notInState(state, psStart, r, i); err != nil {
				return -1, -1, err
			}

			minIsInclusive = true
			state = psMin
			offset = i + 1
		case '(':
			if err := notInState(state, psStart, r, i); err != nil {
				return -1, -1, err
			}

			minIsInclusive = false
			state = psMin
			offset = i + 1
		case ',':
			if err := notInState(state, psMin, r, i); err != nil {
				return -1, -1, err
			}

			minString = strings.TrimSpace(str[offset:i])
			state = psMax
			offset = i + 1
		case ')':
			if err := notInState(state, psMax, r, i); err != nil {
				return -1, -1, err
			}

			maxString = strings.TrimSpace(str[offset:i])
			maxIsInclusive = false
			state = psTimeZone
		case ']':
			if err := notInState(state, psMax, r, i); err != nil {
				return -1, -1, err
			}

			maxString = strings.TrimSpace(str[offset:i])
			maxIsInclusive = true
			state = psTimeZone
		case '@':
			if err := notInState(state, psTimeZone, r, i); err != nil {
				return -1, -1, err
			}

			tzString = strings.TrimSpace(str[i+1:])

			break parserLoop
		}
	}

	if tzString == "" {
		return -1, -1, fmt.Errorf("time zone most be explicit")
	}

	loc, err := time.LoadLocation(tzString)
	if err != nil {
		return -1, -1, err
	}

	minTime, err := time.ParseInLocation(dateTimeFormat, minString, loc)
	if err != nil {
		return -1, -1, err
	}

	maxTime, err := time.ParseInLocation(dateTimeFormat, maxString, loc)
	if err != nil {
		return -1, -1, err
	}

	minUnix := minTime.Unix()
	maxUnix := maxTime.Unix()

	if !minIsInclusive {
		minUnix++
	}

	if !maxIsInclusive {
		maxUnix--
	}

	return minUnix, maxUnix, nil
}

func notInState(state, expected parserState, r rune, pos int) error {
	if state != expected {
		return fmt.Errorf("1:%d: unexpected char '%s'", pos, string(r))
	}

	return nil
}
