package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsRetriablePgError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection exception",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionException},
			want: true,
		},
		{
			name: "connection failure",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			want: true,
		},
		{
			name: "wrapped connection exception",
			err:  fmt.Errorf("add counter: %w", &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist}),
			want: true,
		},
		{
			name: "unique violation",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			want: false,
		},
		{
			name: "undefined table",
			err:  &pgconn.PgError{Code: pgerrcode.UndefinedTable},
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isRetriablePgError(test.err); got != test.want {
				t.Fatalf("isRetriablePgError = %v, want %v", got, test.want)
			}
		})
	}
}
