package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDB(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		host     string
		port     string
		dbname   string
		sslmode  string
		wantErr  bool
	}{
		{
			name:     "invalid connection string",
			username: "user",
			password: "pass",
			host:     "invalid-host",
			port:     "9999",
			dbname:   "testdb",
			sslmode:  "disable",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDB(tt.username, tt.password, tt.host, tt.port, tt.dbname, tt.sslmode)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
			}
		})
	}
}

func TestDB_Close(t *testing.T) {
	// Test that Close method exists
	// We can't test with nil pool as it will panic
	// This test just ensures the method exists
	db := &DB{}
	// We expect this to panic with nil pool, so we test that it does
	assert.Panics(t, func() {
		db.Close()
	})
}
