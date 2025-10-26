package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func InitDB(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'normal' CHECK(role IN ('admin', 'normal')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

	CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_resources_name ON resources(name);
	CREATE INDEX IF NOT EXISTS idx_resources_user_id ON resources(user_id);

	CREATE TABLE IF NOT EXISTS games (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		developer TEXT NOT NULL,
		genres TEXT NOT NULL,
		tags TEXT NOT NULL,
		rating INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
		status TEXT NOT NULL,
		description TEXT NOT NULL,
		my_thoughts TEXT NOT NULL,
		cover_image TEXT NOT NULL,
		explicit INTEGER NOT NULL DEFAULT 0,
		color TEXT NOT NULL,
		percent INTEGER NOT NULL CHECK(percent >= 0 AND percent <= 100),
		bad INTEGER NOT NULL DEFAULT 0,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_games_title ON games(title);
	CREATE INDEX IF NOT EXISTS idx_games_developer ON games(developer);
	CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
	CREATE INDEX IF NOT EXISTS idx_games_rating ON games(rating);
	CREATE INDEX IF NOT EXISTS idx_games_user_id ON games(user_id);

	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		genres TEXT NOT NULL,
		tags TEXT NOT NULL,
		rating INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
		status TEXT NOT NULL,
		description TEXT NOT NULL,
		my_thoughts TEXT NOT NULL,
		cover_image TEXT NOT NULL,
		explicit INTEGER NOT NULL DEFAULT 0,
		color TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
	CREATE INDEX IF NOT EXISTS idx_books_developer ON books(author);
	CREATE INDEX IF NOT EXISTS idx_books_status ON books(status);
	CREATE INDEX IF NOT EXISTS idx_books_rating ON books(rating);
	CREATE INDEX IF NOT EXISTS idx_books_user_id ON books(user_id);

	CREATE TABLE IF NOT EXISTS game_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		game_id INTEGER NOT NULL,
		FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS book_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		book_id INTEGER NOT NULL,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		game_id INTEGER,
		book_id INTEGER,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		CHECK ((game_id IS NOT NULL AND book_id IS NULL) OR (game_id IS NULL AND book_id IS NOT NULL))
	);

	CREATE INDEX IF NOT EXISTS idx_game_links_game_id ON game_links(game_id);
	CREATE INDEX IF NOT EXISTS idx_book_links_book_id ON book_links(book_id);
	CREATE INDEX IF NOT EXISTS idx_comments_game_id ON comments(game_id);
	CREATE INDEX IF NOT EXISTS idx_comments_book_id ON comments(book_id);
	CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
	`

	ctx := context.Background()
	_, err := db.ExecContext(ctx, schema)
	return err
}
