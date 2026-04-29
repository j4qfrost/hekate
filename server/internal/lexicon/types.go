// Package lexicon mirrors the JSON lexicons under lexicons/app/hekate/ as Go
// types. M1 will swap this to generated code once the indigo lexicon
// codegen lands (or its Buf-redesign successor — see ADR 0001). Until then,
// these hand-written types are the contract.
//
// All fields use the JSON name expected by an ATProto record. Optional
// fields are pointers or empty-allowed values per the lexicon's "required"
// list. The CreatedAt field is always required and is the canonical
// ordering key.
package lexicon

// Venue corresponds to lexicons/app/hekate/venue.json (id "app.hekate.venue").
type Venue struct {
	Type          string         `json:"$type"` // always "app.hekate.venue"
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Geo           GeoPoint       `json:"geo"`
	Address       *PostalAddress `json:"address,omitempty"`
	Capacity      *int           `json:"capacity,omitempty"`
	Amenities     []string       `json:"amenities,omitempty"`
	BookingPolicy string         `json:"bookingPolicy"` // "open" | "review"
	Contact       *ContactInfo   `json:"contact,omitempty"`
	CreatedAt     string         `json:"createdAt"`
}

type GeoPoint struct {
	Lat            float64  `json:"lat"`
	Lon            float64  `json:"lon"`
	AltitudeMeters *float64 `json:"altitudeMeters,omitempty"`
}

type PostalAddress struct {
	Text       string `json:"text,omitempty"`
	Locality   string `json:"locality,omitempty"`
	Region     string `json:"region,omitempty"`
	Country    string `json:"country,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
}

type ContactInfo struct {
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Recurrence corresponds to lexicons/app/hekate/recurrence.json.
type Recurrence struct {
	Type                string  `json:"$type"` // always "app.hekate.recurrence"
	Venue               string  `json:"venue"` // at-uri
	RRule               string  `json:"rrule"`
	SlotDurationMinutes int     `json:"slotDurationMinutes"`
	Title               string  `json:"title,omitempty"`
	ValidFrom           string  `json:"validFrom"`
	ValidUntil          *string `json:"validUntil,omitempty"`
	CreatedAt           string  `json:"createdAt"`
}

// Slot corresponds to lexicons/app/hekate/slot.json.
type Slot struct {
	Type        string  `json:"$type"` // always "app.hekate.slot"
	Venue       string  `json:"venue"` // at-uri
	Recurrence  *string `json:"recurrence,omitempty"`
	Start       string  `json:"start"`
	End         string  `json:"end"`
	Status      string  `json:"status"` // "open" | "claimed" | "cancelled"
	ClaimedBy   *string `json:"claimedBy,omitempty"`
	Notes       string  `json:"notes,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

// Event corresponds to lexicons/app/hekate/event.json.
type Event struct {
	Type        string   `json:"$type"` // always "app.hekate.event"
	Slot        string   `json:"slot"`  // at-uri
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Visibility  string   `json:"visibility,omitempty"` // "public" | "unlisted"
	CapacityCap *int     `json:"capacityCap,omitempty"`
	ExternalURL string   `json:"externalUrl,omitempty"`
	CreatedAt   string   `json:"createdAt"`
}

// RSVP corresponds to lexicons/app/hekate/rsvp.json.
type RSVP struct {
	Type       string `json:"$type"` // always "app.hekate.rsvp"
	Event      string `json:"event"` // at-uri
	Status     string `json:"status"` // "going" | "maybe" | "declined"
	GuestCount int    `json:"guestCount,omitempty"`
	Note       string `json:"note,omitempty"`
	CreatedAt  string `json:"createdAt"`
}
