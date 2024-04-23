CREATE TABLE reminders (id INTEGER PRIMARY KEY AUTOINCREMENT,uuid TEXT,sender TEXT,subject TEXT,messageid TEXT,timestamp BIGINT,recurring INTEGER,spec TEXT,status TEXT);
CREATE TABLE settings (name PRIMARY KEY NOT NULL, value TEXT);
