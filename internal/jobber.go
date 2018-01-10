package internal

import (
	"database/sql"

	_ "github.com/lib/pq"
	_ "github.com/mattes/migrate/source/file"
)

var Db *sql.DB

