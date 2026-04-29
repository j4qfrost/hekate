package recurrence

import (
	"testing"
	"time"
)

func TestWeeklyByDayProduces13ThursdaysOver90Days(t *testing.T) {
	// dtstart is itself a Thursday so the weekly schedule produces a clean
	// count: slots at days 0, 7, 14, ..., 84 — 13 slots all strictly before
	// horizon (day 90, exclusive). Picking a non-Thursday dtstart would push
	// the 13th slot past the horizon and yield 12; the indexer's
	// horizon-exclusive semantics are deliberate.
	loc := time.UTC
	dtstart := time.Date(2026, 5, 7, 19, 0, 0, 0, loc) // Thursday
	if dtstart.Weekday() != time.Thursday {
		t.Fatalf("test pre-condition: dtstart must be a Thursday")
	}
	horizon := dtstart.AddDate(0, 0, 90)

	got, err := Expand("FREQ=WEEKLY;BYDAY=TH", dtstart, horizon)
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if len(got) != 13 {
		t.Errorf("expected 13 thursdays in 90 days from a Thursday dtstart, got %d", len(got))
	}
	for _, ts := range got {
		if ts.Weekday() != time.Thursday {
			t.Errorf("expected Thursday, got %s for %s", ts.Weekday(), ts)
		}
		if ts.Hour() != 19 {
			t.Errorf("expected hour 19 inherited from dtstart, got %d", ts.Hour())
		}
	}
}

func TestWeeklyWithByHourOverridesDtstartHour(t *testing.T) {
	loc := time.UTC
	dtstart := time.Date(2026, 5, 4, 9, 30, 0, 0, loc) // Monday
	horizon := dtstart.AddDate(0, 0, 14)

	got, err := Expand("FREQ=WEEKLY;BYDAY=MO;BYHOUR=18", dtstart, horizon)
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 mondays in 14 days, got %d", len(got))
	}
	for _, ts := range got {
		if ts.Hour() != 18 {
			t.Errorf("BYHOUR=18 not applied: got %d", ts.Hour())
		}
	}
}

func TestRejectsNonWeekly(t *testing.T) {
	dtstart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err := Expand("FREQ=DAILY", dtstart, dtstart.AddDate(0, 0, 7))
	if err == nil {
		t.Error("expected error for FREQ=DAILY at v0.1, got nil")
	}
}

func TestRejectsHorizonBeforeStart(t *testing.T) {
	t0 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err := Expand("FREQ=WEEKLY", t0, t0.AddDate(0, 0, -1))
	if err == nil {
		t.Error("expected error for horizon <= dtstart")
	}
}
