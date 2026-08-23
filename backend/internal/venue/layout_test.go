package venue

import "testing"

func TestLayoutValidate(t *testing.T) {
	layout := Layout{Categories: []Category{{Key: "standard", Label: "Standard"}}, Rows: []Row{{Label: "A", Seats: []Seat{{Number: "1", Category: "standard", Column: 1}}}}}
	if err := layout.Validate(); err != nil {
		t.Fatal(err)
	}
	layout.Rows[0].Seats[0].Category = "premium"
	if err := layout.Validate(); err == nil {
		t.Fatal("expected unknown category error")
	}
}
