package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNoSuchUser = errors.New("no user were found")

type User struct {
	UserId         uuid.UUID
	Login          string
	Email          string
	HashedPassword []byte
}

func usersTableSchema() string {
	return `
CREATE TABLE IF NOT EXISTS Users (
	userId UUID PRIMARY KEY,
	login VARCHAR(100) NOT NULL,
	email VARCHAR(255) NOT NULL,
	hashedPassword BYTEA NOT NULL
);`
}

func (tx *Tx) InsertUser(ctx context.Context, user User) error {
	query := "INSERT INTO Users (userId, login, email, hashedPassword) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING"
	_, err := tx.tx.Exec(ctx, query, user.UserId, user.Login, user.Email, user.HashedPassword)
	return err
}

func getUserFromRow(row pgx.Row) (*User, error) {
	var user User
	err := row.Scan(&user.UserId, &user.Login, &user.Email, &user.HashedPassword)
	if err == pgx.ErrNoRows {
		return nil, ErrNoSuchUser
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (tx *Tx) FindUserByLogin(ctx context.Context, login string) (*User, error) {
	query := "SELECT userId, login, email, hashedPassword FROM Users WHERE login = $1"
	return getUserFromRow(tx.tx.QueryRow(ctx, query, login))
}

func (tx *Tx) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	query := "SELECT userId, login, email, hashedPassword FROM Users WHERE email = $1"
	return getUserFromRow(tx.tx.QueryRow(ctx, query, email))
}

func (tx *Tx) FindUserById(ctx context.Context, userId uuid.UUID) (*User, error) {
	query := "SELECT userId, login, email, hashedPassword FROM Users WHERE userId = $1"
	return getUserFromRow(tx.tx.QueryRow(ctx, query, userId))
}

type Profile struct {
	UserId         uuid.UUID
	Name           string
	Surname        string
	PhoneNumber    string
	BirthDay       *time.Time
	CreationTime   *time.Time
	LastUpdateTime *time.Time
}

func profilesTableSchema() string {
	return `
CREATE TABLE IF NOT EXISTS Profiles (
	userId UUID PRIMARY KEY,
	name VARCHAR(100),
	surname VARCHAR(100),
	phoneNumber VARCHAR(20),
	birthDay DATE,
	creationTime TIMESTAMP WITHOUT TIME ZONE,
	lastUpdateTime TIMESTAMP WITHOUT TIME ZONE
);`
}

func (tx *Tx) InsertProfile(ctx context.Context, profile Profile) error {
	query := "INSERT INTO Profiles (userId, name, surname, phoneNumber, birthDay, creationTime, lastUpdateTime) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING"
	_, err := tx.tx.Exec(ctx, query, profile.UserId, profile.Name, profile.Surname, profile.PhoneNumber, profile.BirthDay, profile.CreationTime, profile.LastUpdateTime)
	return err
}

func getProfileFromRow(row pgx.Row) (*Profile, error) {
	var profile Profile
	err := row.Scan(&profile.UserId, &profile.Name, &profile.Surname, &profile.PhoneNumber, &profile.BirthDay, &profile.CreationTime, &profile.LastUpdateTime)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (tx *Tx) FindProfileByUserId(ctx context.Context, userId uuid.UUID) (*Profile, error) {
	query := "SELECT * FROM Profiles WHERE userId = $1"
	return getProfileFromRow(tx.tx.QueryRow(ctx, query, userId))
}

func (tx *Tx) UpdateProfile(ctx context.Context, profile Profile) error {
	query := "UPDATE Profiles SET name = $1, surname = $2, phoneNumber = $3, BirthDay = $4, creationTime = $5, lastUpdateTime = $6 WHERE userId = $7"
	_, err := tx.tx.Exec(ctx, query, profile.Name, profile.Surname, profile.PhoneNumber, profile.BirthDay, profile.CreationTime, profile.LastUpdateTime, profile.UserId)
	return err
}
