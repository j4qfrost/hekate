package index

import "time"

// Venue mirrors app.hekate.venue. Fields named in lower_snake_case in SQL
// are CamelCase here; the sqlc generator will produce its own types from
// SQL that we re-use only at the persistence boundary.
type Venue struct {
	DID            string
	RKey           string
	Name           string
	Description    string
	Lat            float64
	Lon            float64
	AltitudeMeters *float64
	AddressText    string
	Locality       string
	Region         string
	Country        string
	PostalCode     string
	Capacity       *int32
	Amenities      []string
	BookingPolicy  string // "open" | "review"
	ContactEmail   string
	ContactURL     string
	CreatedAt      time.Time
}

type Recurrence struct {
	DID                 string
	RKey                string
	VenueURI            string
	RRule               string
	SlotDurationMinutes int32
	Title               string
	ValidFrom           time.Time
	ValidUntil          *time.Time
	CreatedAt           time.Time
}

type Slot struct {
	DID            string
	RKey           string
	VenueURI       string
	RecurrenceURI  *string
	Start          time.Time
	End            time.Time
	Status         string // "open" | "claimed" | "cancelled"
	ClaimedByURI   *string
	Notes          string
	CreatedAt      time.Time
}

type Event struct {
	DID          string
	RKey         string
	SlotURI      string
	Title        string
	Description  string
	Tags         []string
	Visibility   string // "public" | "unlisted"
	CapacityCap  *int32
	ExternalURL  string
	CreatedAt    time.Time
}

type RSVP struct {
	DID        string
	RKey       string
	EventURI   string
	Status     string // "going" | "maybe" | "declined"
	GuestCount int32
	Note       string
	CreatedAt  time.Time
}
