//go:build cgo

package main

// TestMigrateCommand removed: detectDatabases, getDBVersion, formatDBList, dbInfo
// were removed in Dolt-native pruning. Migration is now handled by bd init --from-jsonl.

// TestFormatDBList removed: formatDBList and dbInfo types were removed.

// TestMigrateRespectsConfigJSON removed: server-mode Dolt backend was removed
// in the nuclear simplification. Migration tests are covered by embedded tests.
