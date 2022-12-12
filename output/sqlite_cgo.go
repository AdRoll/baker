//go:build cgo_sqlite

package output

import (
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	All = append(All, SQLiteDesc, SQLiteRawDesc)
}
