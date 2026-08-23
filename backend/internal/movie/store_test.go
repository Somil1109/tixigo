package movie

import (
	"testing"
	"time"
)

func TestDraftValidation(t *testing.T) {
	draft := Draft{Title: "Film", Description: "Description", PosterURL: "https://example.com/poster", Language: "Hindi", AgeRating: "UA", DurationMinutes: 120, Screenings: []ScreeningInput{{VenueID: "venue", StartsAt: time.Now().Add(time.Hour), Prices: map[string]int{"standard": 20000}}}}
	if err := draft.Validate(); err != nil {
		t.Fatal(err)
	}
	draft.Screenings[0].StartsAt = time.Now().Add(-time.Hour)
	if err := draft.Validate(); err == nil {
		t.Fatal("expected past screening to fail")
	}
}
