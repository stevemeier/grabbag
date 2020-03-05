package lib

import "log"
import "os"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"

func Env_defined(key string) bool {
        _, exists := os.LookupEnv(key)
        return exists
}

func Check_schema(db *sql.DB) bool {
	var err error
	stmt1, err1 := db.Prepare("CREATE TABLE IF NOT EXISTS reminders (" +
	                          "id INTEGER PRIMARY KEY AUTOINCREMENT," +
				  "uuid TEXT," +
				  "sender TEXT," +
				  "subject TEXT," +
				  "messageid TEXT," +
				  "timestamp BIGINT," +
				  "recurring TEXT," +
				  "status TEXT)")
	if err1 != nil {
		log.Fatal(err1)
	}
	defer stmt1.Close()

	_, err = stmt1.Exec()
	if err != nil {
		log.Fatal(err)
	}

	stmt2, err2 := db.Prepare("CREATE TABLE IF NOT EXISTS settings (name PRIMARY KEY NOT NULL, value TEXT)")
	if err2 != nil {
		log.Fatal(err2)
	}
	defer stmt2.Close()

	_, err = stmt2.Exec()
	if err != nil {
		log.Fatal(err)
	}

	return true
}
