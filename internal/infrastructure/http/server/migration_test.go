package server

import (
"testing"
)

func TestMigrationCreatesBothTables(t *testing.T) {
db := setupTestDB(t)
defer db.Close()

// Check what tables exist
rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
if err != nil {
t.Fatalf("failed to query tables: %v", err)
}
defer rows.Close()

tables := make(map[string]bool)
for rows.Next() {
var name string
if err := rows.Scan(&name); err != nil {
t.Fatalf("failed to scan table name: %v", err)
}
tables[name] = true
t.Logf("Found table: %s", name)
}

if !tables["urls"] {
t.Error("urls table not found")
}
if !tables["clicks"] {
t.Error("clicks table not found")
}
}
