package backend

import (
	"testing"

	"github.com/go-faker/faker/v4"
)

// TODO: Better Database Test

func TestDB(t *testing.T) {

	db := NewDatabaseManager()

	for _ = range 100 {
		testDB(t, db)
	}
}

func BenchmarkDB(b *testing.B) {
	db := NewDatabaseManager()

	for b.Loop() {
		testDB(b, db)
	}
}

func testDB(tb testing.TB, db *DatabaseManager) {
	name := faker.Name()
	code, err := db.Login(name)
	if err != nil {
		tb.Logf("Login %s: %v", name, err)
		tb.FailNow()
	}

	uid, err := db.VerifyTransport(code)
	if err != nil {
		tb.Logf("Verify %s: %v", name, err)
		tb.FailNow()
	}

	look, err := db.GetUserByID(uid)
	if err != nil {
		tb.Logf("GetUserByID %s: %v", name, err)
		tb.FailNow()
	}

	if look != name {
		tb.Logf("GetUserByID %s: expected %s, got %s", name, name, look)
		tb.FailNow()
	}
}
