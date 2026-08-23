package seat

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tixigo/tixigo-api/internal/database"
)

func TestConcurrentHoldAllowsOnlyOneCustomer(t *testing.T) {
	connectionString := os.Getenv("TIXIGO_TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("set TIXIGO_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("tixigo_test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE")

	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		t.Fatal(err)
	}
	config.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		_, err := connection.Exec(ctx, `SELECT set_config('search_path',$1,false)`, schema)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = database.ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}

	var firstUser, secondUser, venueID, movieID, screeningID, seatID string
	if err = pool.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,email_verified_at) VALUES('one@example.com','hash','One',now()) RETURNING id::text`).Scan(&firstUser); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,email_verified_at) VALUES('two@example.com','hash','Two',now()) RETURNING id::text`).Scan(&secondUser); err != nil {
		t.Fatal(err)
	}
	layout := `{"categories":[{"key":"standard","label":"Standard"}],"rows":[{"label":"A","seats":[{"number":"1","category":"standard"}]}]}`
	if err = pool.QueryRow(ctx, `INSERT INTO venues(name,address,city,layout,created_by) VALUES('Cinema','Street','Mumbai',$1,$2) RETURNING id::text`, layout, firstUser).Scan(&venueID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO movies(title,description,poster_url,language,duration_minutes,age_rating,status,organiser_id) VALUES('Film','Description','https://example.com/poster','Hindi',120,'UA','published',$1) RETURNING id::text`, firstUser).Scan(&movieID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO screenings(movie_id,venue_id,starts_at) VALUES($1,$2,now()+interval '2 days') RETURNING id::text`, movieID, venueID).Scan(&screeningID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO screening_seats(screening_id,seat_key,row_label,seat_number,category,price_paise) VALUES($1,'A1','A','1','standard',25000) RETURNING id::text`, screeningID).Scan(&seatID); err != nil {
		t.Fatal(err)
	}

	store := NewStore(pool)
	var successes atomic.Int32
	var group sync.WaitGroup
	for _, userID := range []string{firstUser, secondUser} {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := store.Hold(ctx, screeningID, userID, []string{seatID}); err == nil {
				successes.Add(1)
			}
		}()
	}
	group.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful holds = %d, want 1", successes.Load())
	}
}
