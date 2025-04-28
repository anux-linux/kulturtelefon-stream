package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store interface {
	CreateIcecastMount(mount IcecastMount) error
	DeleteIcecastMount(mountName string) (IcecastMount, error)
	GetIcecastMount(mountName string) (IcecastMount, error)
	GetIcecastMounts() ([]IcecastMount, error)
	UpdateIcecastMount(mount IcecastMount) error

	SaveUser(username, password string) error
	GetUser(username string) (string, error)
	DeleteUser(username string) error
	GetUserByToken(token string) (string, error)
	GetTokenByUser(username string) (string, error)

	SaveToken(username, token string) error
}

type SqliteStorage struct {
	db *sql.DB
}

func NewSqliteStore(config *Config) (*SqliteStorage, error) {

	db, err := sql.Open("sqlite", config.DbFile)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error opening database: %v", err), FatalLog)
		return nil, err
	}
	initDBerr := initDb(db, config)
	if initDBerr != nil {
		logWithCaller(fmt.Sprintf("Error initializing database: %v", err), FatalLog)
		return nil, err
	}

	return &SqliteStorage{
		db: db,
	}, nil
}

func initDb(db *sql.DB, config *Config) error {
	logWithCaller("Initializing database", InfoLog)
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS icecast_mounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mount_name TEXT UNIQUE NOT NULL,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		public INTEGER NOT NULL,
		stream_name TEXT NOT NULL,
		stream_description TEXT NOT NULL,
		template_type TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_mount_name ON icecast_mounts (mount_name);

	CREATE TABLE IF NOT EXISTS token (
		token_hash TEXT UNIQUE NOT NULL,
		user_id INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		icecast_mount TEXT UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error creating database table: %v", err), FatalLog)
		return err
	}

	err = createAdminUser(db, config.AdminUsername, config.AdminPassword)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error creating admin: %v", err), FatalLog)
		return err
	}

	// Set the maximum number of open connections to 1
	db.SetMaxOpenConns(2)
	// Set the maximum number of idle connections to 1
	db.SetMaxIdleConns(2)
	// Set the maximum lifetime of a connection to 5 minutes
	db.SetConnMaxLifetime(1 * 60 * 1000) // 1 minutes

	return nil
}

func createAdminUser(db *sql.DB, username, password string) error {
	logWithCaller("Creating admin user", InfoLog)

	hashedPassword, err := getHashedPassword(password)
	if err != nil {
		logWithCaller("Error hashing password: "+err.Error(), FatalLog)
		return err
	}

	logWithCaller("Inserting admin user. Will overwrite existing admin user.", InfoLog)
	_, err = db.Exec(`
	INSERT INTO users (username, password) 
	VALUES (?, ?)
	ON CONFLICT(username) DO UPDATE SET password = excluded.password;`, username, hashedPassword)
	if err != nil {
		logWithCaller(fmt.Sprintf("Databas error creating admin: %v", err), FatalLog)
		return err
	}
	logWithCaller("Admin user created", InfoLog)
	return nil
}

func (s *SqliteStorage) CreateIcecastMount(mount IcecastMount) error {
	stmt, err := s.db.Prepare(`
	INSERT INTO icecast_mounts (mount_name, username, password, public, stream_name, stream_description, template_type)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error creating icecast_mounts prepared statement: %v", err), FatalLog)
		return err
	}

	result, err := stmt.Exec(mount.MountName, mount.Username, mount.Password, mount.Public, mount.StreamName, mount.StreamDescription, mount.TemplateType)

	if err != nil {
		logWithCaller(fmt.Sprintf("Database error excecutiong icecast_mounts prepared statement: %v", err), FatalLog)
		return err
	}
	lastID, err := result.LastInsertId()
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error getting last insert ID on icecast_mounts prepared statement: %v", err), FatalLog)
		return err
	}

	logWithCaller(fmt.Sprintf("Inserted row with ID: %d", lastID), InfoLog)

	defer stmt.Close()

	return nil
}
func (s *SqliteStorage) DeleteIcecastMount(mountName string) (IcecastMount, error) {
	logWithCaller(fmt.Sprintf("Deleting mount from Database: %s", mountName), InfoLog)
	mount, err := s.GetIcecastMount(mountName)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error getting mount: %s %v", mountName, err), FatalLog)
		return IcecastMount{}, err
	}

	stmt, err := s.db.Prepare(`
	DELETE FROM icecast_mounts
	WHERE mount_name = $1
	`)

	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %v", err), FatalLog)
		return IcecastMount{}, err
	}
	defer stmt.Close()
	result, err := stmt.Exec(mountName)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error exceuting statement: %v", err), FatalLog)
		return IcecastMount{}, err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		logWithCaller(fmt.Sprintf("Error getting affected rows: %v", err), FatalLog)
		return IcecastMount{}, err
	}
	if affectedRows != 1 {
		logWithCaller(fmt.Sprintf("Affected rows %d for: %s ", affectedRows, mountName), FatalLog)
		return IcecastMount{}, fmt.Errorf("affected rows %d for: %s", affectedRows, mountName)
	}
	logWithCaller(fmt.Sprintf("Deleted mount: %s", mountName), InfoLog)
	logWithCaller(fmt.Sprintf("Affected rows: %d", affectedRows), DebugLog)

	return mount, nil
}
func (s *SqliteStorage) GetIcecastMount(mountName string) (IcecastMount, error) {
	logWithCaller(fmt.Sprintf("Getting mount from Database: %s", mountName), InfoLog)

	stmt, err := s.db.Prepare(`
	SELECT mount_name, username, password, public, stream_name, stream_description, template_type
	FROM icecast_mounts
	WHERE mount_name = $1
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %v", err), FatalLog)
		return IcecastMount{}, err
	}
	defer stmt.Close()
	var mount IcecastMount
	err = stmt.QueryRow(mountName).Scan(&mount.MountName, &mount.Username, &mount.Password, &mount.Public, &mount.StreamName, &mount.StreamDescription, &mount.TemplateType)
	if err != nil {
		if err == sql.ErrNoRows {
			logWithCaller(fmt.Sprintf("No rows found for mount name: %s", mountName), DebugLog)
			return IcecastMount{}, err
		}
		return IcecastMount{}, err
	}
	return mount, nil
}
func (s *SqliteStorage) GetIcecastMounts() ([]IcecastMount, error) {
	logWithCaller("Getting all mounts from Database", InfoLog)
	stmt, err := s.db.Prepare(`
	SELECT mount_name, username, password, public, stream_name, stream_description, template_type
	FROM icecast_mounts
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %v", err), FatalLog)

		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing statement: %v", err), FatalLog)

		return nil, err
	}
	defer rows.Close()
	var mounts []IcecastMount
	for rows.Next() {
		var mount IcecastMount
		err = rows.Scan(&mount.MountName, &mount.Username, &mount.Password, &mount.Public, &mount.StreamName, &mount.StreamDescription, &mount.TemplateType)
		if err != nil {
			logWithCaller(fmt.Sprintf("Error scanning row: %v", err), FatalLog)

			return nil, err
		}
		mounts = append(mounts, mount)
	}
	if err = rows.Err(); err != nil {
		logWithCaller(fmt.Sprintf("Error iterating rows: %v", err), FatalLog)

		return nil, err
	}
	logWithCaller(fmt.Sprintf("Found %d mounts", len(mounts)), InfoLog)

	return mounts, nil
}
func (s *SqliteStorage) UpdateIcecastMount(mount IcecastMount) error {
	logWithCaller(fmt.Sprintf("Updating mount in Database: %s", mount.MountName), InfoLog)

	stmt, err := s.db.Prepare(`
	UPDATE icecast_mounts
	SET username = $1,
	password = $2,
	public = $3,
	stream_name = $4,
	stream_description = $5,
	template_type = $6
	WHERE mount_name = $7
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %v", err), FatalLog)
		return err
	}
	defer stmt.Close()
	result, err := stmt.Exec(mount.Username, mount.Password, mount.Public, mount.StreamName, mount.StreamDescription, mount.TemplateType, mount.MountName)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing statement: %v", err), FatalLog)
		return err
	}

	affecttedRows, err := result.RowsAffected()
	if err != nil {
		logWithCaller(fmt.Sprintf("Error getting affected rows: %v", err), FatalLog)

		return err
	}
	logWithCaller(fmt.Sprintf("Affected rows: %d", affecttedRows), DebugLog)

	if affecttedRows != 1 {
		logWithCaller(fmt.Sprintf("Affected rows %d for: %s", affecttedRows, mount.MountName), FatalLog)
		return fmt.Errorf("affected rows %d for: %s", affecttedRows, mount.MountName)
	}

	logWithCaller(fmt.Sprintf("Updated mount in database: %s", mount.MountName), InfoLog)

	return nil
}

func (s *SqliteStorage) SaveUser(username, hashedPassword string) error {
	logWithCaller(fmt.Sprintf("Saving user to database: %s", username), InfoLog)
	stmt, err := s.db.Prepare(`
	INSERT INTO users (username, password)
	VALUES ($1, $2)
	ON CONFLICT(username) DO UPDATE SET password = excluded.password;
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %v", err), FatalLog)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(username, hashedPassword)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing statement: %v", err), FatalLog)
		return err
	}
	logWithCaller(fmt.Sprintf("Inserted user: %s", username), InfoLog)

	return nil
}

func (s *SqliteStorage) GetUser(username string) (string, error) {
	logWithCaller(fmt.Sprintf("Getting user from database: %s", username), InfoLog)
	stmt, err := s.db.Prepare(`
	SELECT password
	FROM users
	WHERE username = $1
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %v", err), FatalLog)
		return "", err
	}
	defer stmt.Close()
	var hashedPassword string
	err = stmt.QueryRow(username).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			logWithCaller(fmt.Sprintf("No rows found for username: %s", username), DebugLog)
			return "", err
		}
		return "", err
	}
	return hashedPassword, nil
}
func (s *SqliteStorage) DeleteUser(username string) error {

	tokenDelStmt, err := s.db.Prepare(`
	DELETE FROM token
	WHERE user_id = (SELECT id FROM users WHERE username = $1)
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %s", err.Error()), FatalLog)
		return err
	}
	defer tokenDelStmt.Close()
	_, err = tokenDelStmt.Exec(username)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing statement: %s", err.Error()), FatalLog)
		return err
	}

	userDelStmt, err := s.db.Prepare(`
	DELETE FROM users
	WHERE username = $1
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %s", err.Error()), FatalLog)
		return err
	}
	defer userDelStmt.Close()
	_, err = userDelStmt.Exec(username)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing statement: %s", err.Error()), FatalLog)
		return err
	}
	return nil
}
func (s *SqliteStorage) GetUserByToken(tokenHash string) (string, error) {
	logWithCaller("Getting user by token from database", InfoLog)
	stmt, err := s.db.Prepare(`
	SELECT username
	FROM users
	WHERE id = (SELECT user_id FROM token WHERE token_hash = $1)
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %s", err.Error()), FatalLog)
		return "", err
	}
	defer stmt.Close()
	var username string
	err = stmt.QueryRow(tokenHash).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			logWithCaller(fmt.Sprintf("No rows found for token: %s", tokenHash), FatalLog)
			return "", err
		}
		logWithCaller(fmt.Sprintf("Error scanning row: %s", err.Error()), WarnLog)
		return "", err
	}
	logWithCaller(fmt.Sprintf("Found username: %s", username), DebugLog)
	return username, nil
}

func (s *SqliteStorage) GetTokenByUser(username string) (string, error) {
	logWithCaller(fmt.Sprintf("Getting token from database for user: %s", username), InfoLog)
	stmt, err := s.db.Prepare(`
	SELECT token_hash
	FROM token
	WHERE username = $1
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %s", err.Error()), FatalLog)

		return "", err
	}
	defer stmt.Close()
	var token_hash string
	err = stmt.QueryRow(username).Scan(&token_hash)
	if err != nil {
		if err == sql.ErrNoRows {
			logWithCaller(fmt.Sprintf("No rows found for username: %s", username), FatalLog)

			return "", err
		}
		return "", err
	}
	return token_hash, nil
}

// SaveToken saves the token hash for a user. Token must be a hash.
func (s *SqliteStorage) SaveToken(username, token_hash string) error {
	insertToken, err := s.db.Prepare(`
	INSERT INTO token (user_id, token_hash)
	VALUES ((SELECT id FROM users WHERE username = $1) ,$2)
	`)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error preparing statement: %s", err.Error()), WarnLog)
		return err
	}
	defer insertToken.Close()

	_, err = insertToken.Exec(username, token_hash)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing statement: %s", err.Error()), WarnLog)
		return err
	}

	logWithCaller(fmt.Sprintf("Inserted hash %s for user: %s", token_hash, username), DebugLog)

	return nil
}
