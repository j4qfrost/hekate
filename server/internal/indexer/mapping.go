package indexer

import (
	"fmt"
	"time"

	"github.com/j4qfrost/hekate/server/internal/firehose"
	"github.com/j4qfrost/hekate/server/internal/index"
	"github.com/j4qfrost/hekate/server/internal/lexicon"
)

// The lexicon.* types mirror the on-PDS record shape (string timestamps,
// ATProto field names); the index.* types are the persistence boundary
// (time.Time, DB column names). These mappers are the one place that
// translation happens. DID and RKey come from the firehose envelope, never
// the record body.

// parseTime parses an ATProto RFC3339 datetime. The field name is included in
// the error so a malformed record points at the offending field.
func parseTime(field, s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: bad RFC3339 time %q: %w", field, s, err)
	}
	return t, nil
}

// i32ptr narrows an optional lexicon int to the int32 the DB columns use.
func i32ptr(p *int) *int32 {
	if p == nil {
		return nil
	}
	v := int32(*p)
	return &v
}

func venueToIndex(ev firehose.Event, rec lexicon.Venue) (index.Venue, error) {
	createdAt, err := parseTime("createdAt", rec.CreatedAt)
	if err != nil {
		return index.Venue{}, err
	}
	v := index.Venue{
		DID:            ev.DID,
		RKey:           ev.RKey,
		Name:           rec.Name,
		Description:    rec.Description,
		Lat:            rec.Geo.Lat,
		Lon:            rec.Geo.Lon,
		AltitudeMeters: rec.Geo.AltitudeMeters,
		Capacity:       i32ptr(rec.Capacity),
		Amenities:      rec.Amenities,
		BookingPolicy:  rec.BookingPolicy,
		CreatedAt:      createdAt,
	}
	if rec.Address != nil {
		v.AddressText = rec.Address.Text
		v.Locality = rec.Address.Locality
		v.Region = rec.Address.Region
		v.Country = rec.Address.Country
		v.PostalCode = rec.Address.PostalCode
	}
	if rec.Contact != nil {
		v.ContactEmail = rec.Contact.Email
		v.ContactURL = rec.Contact.URL
	}
	return v, nil
}

func recurrenceToIndex(ev firehose.Event, rec lexicon.Recurrence) (index.Recurrence, error) {
	createdAt, err := parseTime("createdAt", rec.CreatedAt)
	if err != nil {
		return index.Recurrence{}, err
	}
	validFrom, err := parseTime("validFrom", rec.ValidFrom)
	if err != nil {
		return index.Recurrence{}, err
	}
	r := index.Recurrence{
		DID:                 ev.DID,
		RKey:                ev.RKey,
		VenueURI:            rec.Venue,
		RRule:               rec.RRule,
		SlotDurationMinutes: int32(rec.SlotDurationMinutes),
		Title:               rec.Title,
		ValidFrom:           validFrom,
		CreatedAt:           createdAt,
	}
	if rec.ValidUntil != nil {
		validUntil, err := parseTime("validUntil", *rec.ValidUntil)
		if err != nil {
			return index.Recurrence{}, err
		}
		r.ValidUntil = &validUntil
	}
	return r, nil
}

func slotToIndex(ev firehose.Event, rec lexicon.Slot) (index.Slot, error) {
	createdAt, err := parseTime("createdAt", rec.CreatedAt)
	if err != nil {
		return index.Slot{}, err
	}
	start, err := parseTime("start", rec.Start)
	if err != nil {
		return index.Slot{}, err
	}
	end, err := parseTime("end", rec.End)
	if err != nil {
		return index.Slot{}, err
	}
	return index.Slot{
		DID:           ev.DID,
		RKey:          ev.RKey,
		VenueURI:      rec.Venue,
		RecurrenceURI: rec.Recurrence,
		Start:         start,
		End:           end,
		Status:        rec.Status,
		ClaimedByURI:  rec.ClaimedBy,
		Notes:         rec.Notes,
		CreatedAt:     createdAt,
	}, nil
}

func eventToIndex(ev firehose.Event, rec lexicon.Event) (index.Event, error) {
	createdAt, err := parseTime("createdAt", rec.CreatedAt)
	if err != nil {
		return index.Event{}, err
	}
	// visibility is optional in the lexicon; its default is "public", which
	// is also what the event_records.visibility CHECK constraint expects.
	visibility := rec.Visibility
	if visibility == "" {
		visibility = "public"
	}
	return index.Event{
		DID:         ev.DID,
		RKey:        ev.RKey,
		SlotURI:     rec.Slot,
		Title:       rec.Title,
		Description: rec.Description,
		Tags:        rec.Tags,
		Visibility:  visibility,
		CapacityCap: i32ptr(rec.CapacityCap),
		ExternalURL: rec.ExternalURL,
		CreatedAt:   createdAt,
	}, nil
}

func rsvpToIndex(ev firehose.Event, rec lexicon.RSVP) (index.RSVP, error) {
	createdAt, err := parseTime("createdAt", rec.CreatedAt)
	if err != nil {
		return index.RSVP{}, err
	}
	return index.RSVP{
		DID:        ev.DID,
		RKey:       ev.RKey,
		EventURI:   rec.Event,
		Status:     rec.Status,
		GuestCount: int32(rec.GuestCount),
		Note:       rec.Note,
		CreatedAt:  createdAt,
	}, nil
}
