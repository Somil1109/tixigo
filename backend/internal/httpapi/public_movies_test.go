package httpapi

import (
	"testing"
	"time"
)

func TestIndiaDate(t *testing.T) {
	start, end, err := indiaDate("2026-08-23")
	if err != nil {
		t.Fatal(err)
	}
	if end.Sub(*start) != 24*time.Hour {
		t.Fatalf("duration = %v", end.Sub(*start))
	}
	if start.Location().String() != "Asia/Kolkata" {
		t.Fatalf("location = %s", start.Location())
	}
}
