package lib

import "log"
import "regexp"
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
				  "recurring INTEGER," +
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

func Is_uuid(input string) bool {
	// Regex from:
	// https://github.com/ramsey/uuid/blob/c141cdc8dafa3e506f69753b692f6662b46aa933/src/Uuid.php#L96
	match, _ := regexp.MatchString("^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$", input)
	return match
}

func Disable_reminder(db *sql.DB, addr string) bool {
	stmt1, err1 := db.Prepare("UPDATE reminders SET recurring = 0 WHERE uuid = ?")
	if err1 != nil {
		log.Fatal(err1)
	}
	defer stmt1.Close()

	_, err := stmt1.Exec(addr)
	return err == nil
}
