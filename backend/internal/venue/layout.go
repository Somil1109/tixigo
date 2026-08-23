package venue

import (
	"errors"
	"fmt"
	"strings"
)

type Category struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}
type Seat struct {
	Number   string `json:"number"`
	Category string `json:"category"`
	Column   int    `json:"column"`
}
type Row struct {
	Label string `json:"label"`
	Seats []Seat `json:"seats"`
}
type Layout struct {
	Categories []Category `json:"categories"`
	Rows       []Row      `json:"rows"`
}

func (layout Layout) Validate() error {
	if len(layout.Categories) == 0 || len(layout.Rows) == 0 {
		return errors.New("layout requires categories and rows")
	}
	categories := map[string]struct{}{}
	for _, category := range layout.Categories {
		key := strings.TrimSpace(category.Key)
		if key == "" || strings.TrimSpace(category.Label) == "" {
			return errors.New("each category requires key and label")
		}
		if _, ok := categories[key]; ok {
			return fmt.Errorf("duplicate category %q", key)
		}
		categories[key] = struct{}{}
	}
	seen := map[string]struct{}{}
	count := 0
	for _, row := range layout.Rows {
		if strings.TrimSpace(row.Label) == "" || len(row.Seats) == 0 {
			return errors.New("each row requires a label and seats")
		}
		for _, seat := range row.Seats {
			count++
			if count > 1000 {
				return errors.New("layout cannot exceed 1000 seats")
			}
			if strings.TrimSpace(seat.Number) == "" || seat.Column < 1 {
				return errors.New("each seat requires number and positive column")
			}
			if _, ok := categories[seat.Category]; !ok {
				return fmt.Errorf("seat %s%s uses unknown category", row.Label, seat.Number)
			}
			key := row.Label + ":" + seat.Number
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate seat %s%s", row.Label, seat.Number)
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}
