package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type URL struct {
	ID        int
	Url       string
	CreatedAt time.Time
}

type FileExt struct {
	ID        int
	Extension string
}

type CharFrequency struct {
	URLID     int
	FileExtID int
	Char      string
	Frequency int
}

func initDB(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", file+"?_foreign_keys=1")
	if err != nil {
		return nil, err
	}

	createTables := `
    CREATE TABLE IF NOT EXISTS urls (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT UNIQUE NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS file_extensions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        extension TEXT UNIQUE NOT NULL
    );
    
    CREATE TABLE IF NOT EXISTS char_frequencies (
        url_id INTEGER NOT NULL,
        extension_id INTEGER NOT NULL,
        character TEXT NOT NULL,
        frequency INTEGER NOT NULL,
        PRIMARY KEY (url_id, extension_id, character),
        FOREIGN KEY (url_id) REFERENCES urls(id),
        FOREIGN KEY (extension_id) REFERENCES file_extensions(id)
    );`

	_, err = db.Exec(createTables)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func addURL(db *sql.DB, url string) (int64, error) {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO urls (url) VALUES (?)",
		url,
	)
	if err != nil {
		return 0, err
	}

	var existingID int64
	err = db.QueryRow("SELECT id FROM urls WHERE url = ?", url).Scan(&existingID)
	return existingID, err

}

func addExtension(db *sql.DB, ext string) (int64, error) {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO file_extensions (extension) VALUES (?)",
		ext,
	)
	if err != nil {
		return 0, err
	}

	var existingID int64
	err = db.QueryRow("SELECT id FROM file_extensions WHERE extension = ?", ext).Scan(&existingID)
	return existingID, err

}

func addCharFrequency(db *sql.DB, urlID, extID int64, char string, freq int) error {
	_, err := db.Exec(
		`INSERT INTO char_frequencies (url_id, extension_id, character, frequency)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(url_id, extension_id, character) 
        DO UPDATE SET frequency = frequency + excluded.frequency`,
		urlID, extID, char, freq,
	)
	return err
}

func getStatistics(db *sql.DB, url, ext string) (map[rune]int, error) {
	rows, err := db.Query(
		`SELECT character, frequency 
		FROM char_frequencies f
		JOIN file_extensions e 
		ON f.extension_id=e.ID 
		JOIN urls u
		ON f.url_id=u.ID 
		WHERE u.url= ? AND e.extension = ?`,
		url, ext,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[rune]int)
	for rows.Next() {
		var char string
		var freq int
		err = rows.Scan(&char, &freq)
		if err != nil {
			return nil, err
		}
		stats[rune(char[0])] = freq
	}
	return stats, nil
}

func deleteStatistics(db *sql.DB, urlID int64, extID int64) error {
	_, err := db.Exec(
		"DELETE FROM file_extensions WHERE url_id=? and extension_id=?",
		urlID, extID,
	)
	return err
}

func saveStatistics(db *sql.DB, repoURL string, ext string, stat map[rune]int) error {
	urlID, err := addURL(db, repoURL)
	if err != nil {
		return err
	}

	extID, err := addExtension(db, ext)
	if err != nil {
		return err
	}

	deleteStatistics(db, urlID, extID)

	for char, freq := range stat {
		err = addCharFrequency(db, urlID, extID, string(char), freq)
		if err != nil {
			fmt.Print(err)
			return err
		}
	}
	return nil
}
