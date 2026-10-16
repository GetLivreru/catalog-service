package rcpostgres

import (
	"testing"

	"github.com/uptrace/bun/migrate"
)

func TestMigrationVersion(t *testing.T) {
	tests := []struct {
		name       string
		migrations migrate.MigrationSlice
		want       int64
		wantErr    bool
	}{
		{name: "empty database"},
		{name: "leading zeros", migrations: migrate.MigrationSlice{{Name: "001"}}, want: 1},
		{name: "unordered numeric versions", migrations: migrate.MigrationSlice{{Name: "2"}, {Name: "10"}, {Name: "1"}}, want: 10},
		{name: "invalid version", migrations: migrate.MigrationSlice{{Name: "invalid"}}, wantErr: true},
		{name: "overflow", migrations: migrate.MigrationSlice{{Name: "9223372036854775808"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := migrationVersion(tt.migrations)
			if (err != nil) != tt.wantErr {
				t.Fatalf("migrationVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("migrationVersion() = %d, want %d", got, tt.want)
			}
		})
	}
}
