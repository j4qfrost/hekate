// Package recurrence expands app.hekate.recurrence records into materialised
// app.hekate.slot rows up to a configured horizon.
//
// M0 ships only the interface + a hand-rolled BYDAY=<day> WEEKLY validator
// good enough for tests. M1 swaps in github.com/teambition/rrule-go which
// implements full RFC 5545. This stub deliberately does NOT pretend to
// implement the full standard — see the rule.parse() comment.
package recurrence

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Expand returns the set of slot start times produced by the given RRULE
// string and DTSTART, capped at horizon. End time is computed by the caller
// from slotDurationMinutes.
//
// At v0.1 only FREQ=WEEKLY with optional BYDAY is supported. M1 widens this
// via rrule-go.
func Expand(rrule string, dtstart time.Time, horizon time.Time) ([]time.Time, error) {
	if !horizon.After(dtstart) {
		return nil, fmt.Errorf("recurrence: horizon %s is not after dtstart %s", horizon, dtstart)
	}
	r, err := parse(rrule)
	if err != nil {
		return nil, err
	}
	if r.freq != "WEEKLY" {
		return nil, fmt.Errorf("recurrence: only FREQ=WEEKLY is supported in v0.1 (got %q); rrule-go lands with M1", r.freq)
	}
	var out []time.Time
	for cur := dtstart; cur.Before(horizon); cur = cur.AddDate(0, 0, 1) {
		if !r.matchesDay(cur.Weekday()) {
			continue
		}
		// Match BYHOUR if specified, else inherit from dtstart.
		hr, mn := dtstart.Hour(), dtstart.Minute()
		if r.byHour >= 0 {
			hr = r.byHour
		}
		next := time.Date(cur.Year(), cur.Month(), cur.Day(), hr, mn, 0, 0, cur.Location())
		if next.Before(dtstart) || !next.Before(horizon) {
			continue
		}
		out = append(out, next)
	}
	return out, nil
}

type rule struct {
	freq   string
	byDay  map[time.Weekday]bool
	byHour int // -1 if unset
}

func (r *rule) matchesDay(d time.Weekday) bool {
	if len(r.byDay) == 0 {
		return true
	}
	return r.byDay[d]
}

var dayCodes = map[string]time.Weekday{
	"SU": time.Sunday,
	"MO": time.Monday,
	"TU": time.Tuesday,
	"WE": time.Wednesday,
	"TH": time.Thursday,
	"FR": time.Friday,
	"SA": time.Saturday,
}

func parse(s string) (*rule, error) {
	r := &rule{byHour: -1, byDay: map[time.Weekday]bool{}}
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("recurrence: malformed RRULE part %q", part)
		}
		switch strings.ToUpper(k) {
		case "FREQ":
			r.freq = strings.ToUpper(v)
		case "BYDAY":
			for _, code := range strings.Split(v, ",") {
				wd, ok := dayCodes[strings.ToUpper(strings.TrimSpace(code))]
				if !ok {
					return nil, fmt.Errorf("recurrence: unknown BYDAY code %q", code)
				}
				r.byDay[wd] = true
			}
		case "BYHOUR":
			var h int
			if _, err := fmt.Sscanf(v, "%d", &h); err != nil || h < 0 || h > 23 {
				return nil, fmt.Errorf("recurrence: bad BYHOUR %q", v)
			}
			r.byHour = h
		default:
			// Other RRULE parts (UNTIL, COUNT, INTERVAL, BYMONTH, etc.)
			// are silently accepted-and-ignored at v0.1; rrule-go in M1
			// handles the rest.
		}
	}
	if r.freq == "" {
		return nil, errors.New("recurrence: RRULE missing FREQ")
	}
	return r, nil
}
