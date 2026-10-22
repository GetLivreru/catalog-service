package rcpostgres

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
)

type affectedRows int64

func (r affectedRows) LastInsertId() (int64, error) { return 0, nil }
func (r affectedRows) RowsAffected() (int64, error) { return int64(r), nil }

func TestUpdateErr(t *testing.T) {
	dbErr := errors.New("database unavailable")
	for _, tt := range []struct {
		name           string
		result         sql.Result
		err, errorWant error
	}{
		{"updated", affectedRows(1), nil, nil},
		{"missing", affectedRows(0), nil, entity.ErrNotFound},
		{"wrapped no rows", nil, fmt.Errorf("update: %w", sql.ErrNoRows), entity.ErrNotFound},
		{"database failure", nil, dbErr, dbErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpdateErr(tt.result, tt.err); !errors.Is(got, tt.errorWant) {
				t.Fatalf("got %v, want %v", got, tt.errorWant)
			}
		})
	}
}

func TestDeleteErr(t *testing.T) {
	dbErr := errors.New("foreign key violation")
	for _, tt := range []struct {
		name      string
		err, want error
	}{
		{"deleted", nil, nil},
		{"wrapped no rows", fmt.Errorf("delete: %w", sql.ErrNoRows), nil},
		{"database failure", dbErr, dbErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeleteErr(tt.err); !errors.Is(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
