package backend

// Simple in memory DB for testing

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
)

type TransportSchema struct {
	user    uuid.UUID
	expires time.Time
}

type TransportDatabase struct {
	mu    sync.Mutex
	codes map[uuid.UUID]TransportSchema
}

func NewTransportDatabase() *TransportDatabase {
	return &TransportDatabase{
		codes: make(map[uuid.UUID]TransportSchema),
	}
}

type UserDatabase struct {
	mu    sync.Mutex
	users map[uuid.UUID]string
}

func NewUserDatabase() *UserDatabase {
	return &UserDatabase{
		users: make(map[uuid.UUID]string),
	}
}

type DatabaseManager struct {
	Users     *UserDatabase
	Transport *TransportDatabase
}

func NewDatabaseManager() *DatabaseManager {
	return &DatabaseManager{
		Users:     NewUserDatabase(),
		Transport: NewTransportDatabase(),
	}
}

func (db *DatabaseManager) GetUser(name string) (uuid.UUID, error) {
	db.Users.mu.Lock()
	defer db.Users.mu.Unlock()
	uid := uuid.Nil
	for id, u := range db.Users.users {
		if u == name {
			uid = id
			return uid, nil
		}
	}

	uid, err := uuid.NewV7()
	if err != nil {
		log.Println("UID v7", err)
		return uuid.Nil, err
	}
	db.Users.users[uid] = name

	return uid, nil
}

func (db *DatabaseManager) GetUserByID(uid uuid.UUID) (string, error) {
	db.Users.mu.Lock()
	defer db.Users.mu.Unlock()
	name, ok := db.Users.users[uid]
	if !ok {
		return "", errors.New("user not found")
	}
	return name, nil
}

func (db *DatabaseManager) NewTransport(uid uuid.UUID) (uuid.UUID, error) {
	db.pruneTransport()

	db.Transport.mu.Lock()
	defer db.Transport.mu.Unlock()

	code, err := uuid.NewV4()
	if err != nil {
		return uuid.Nil, err
	}

	db.Transport.codes[code] = TransportSchema{
		user:    uid,
		expires: time.Now().Add(time.Minute * 5),
	}

	return code, nil
}

func (db *DatabaseManager) pruneTransport() {
	db.Transport.mu.Lock()
	defer db.Transport.mu.Unlock()
	for id, transport := range db.Transport.codes {
		if transport.expires.Before(time.Now()) {
			delete(db.Transport.codes, id)
		}
	}
}

func (db *DatabaseManager) VerifyTransport(code uuid.UUID) (uuid.UUID, error) {
	db.pruneTransport()
	db.Transport.mu.Lock()
	defer db.Transport.mu.Unlock()
	s, ok := db.Transport.codes[code]
	if !ok {
		return uuid.Nil, errors.New("transport not found")
	}

	uid := s.user
	delete(db.Transport.codes, code)

	return uid, nil
}

func (db *DatabaseManager) Login(name string) (uuid.UUID, error) {
	uid, err := db.GetUser(name)
	if err != nil {
		return uuid.Nil, err
	}
	code, err := db.NewTransport(uid)
	if err != nil {
		return uuid.Nil, err
	}
	return code, nil
}
