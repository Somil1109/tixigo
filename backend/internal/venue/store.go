package venue

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Venue struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	Timezone string `json:"timezone"`
	Layout   Layout `json:"layout"`
}
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool} }
func (s *Store) Create(ctx context.Context, v Venue, createdBy string) (Venue, error) {
	raw, _ := json.Marshal(v.Layout)
	err := s.pool.QueryRow(ctx, `INSERT INTO venues(name,address,city,timezone,layout,created_by) VALUES($1,$2,$3,'Asia/Kolkata',$4,$5) RETURNING id::text,timezone`, v.Name, v.Address, v.City, raw, createdBy).Scan(&v.ID, &v.Timezone)
	return v, err
}
func (s *Store) List(ctx context.Context) ([]Venue, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text,name,address,city,timezone,layout FROM venues ORDER BY city,name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Venue{}
	for rows.Next() {
		var v Venue
		var raw []byte
		if err := rows.Scan(&v.ID, &v.Name, &v.Address, &v.City, &v.Timezone, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &v.Layout); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
