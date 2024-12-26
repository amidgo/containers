package postgrescontainer

import (
	"database/sql"
	"log"
	"sync"
	"testing"

	"github.com/amidgo/containers"
)

type Reuse struct {
	mu sync.Mutex
	db *sql.DB

	containerTerm func()
}

var global Reuse

func GlobalReuse() *Reuse {
	return &global
}

func ReuseSingleForTesting(t *testing.T, p *Reuse, migrations Migrations, initialQueries ...string) *sql.DB {
	containers.SkipDisabled(t)

	db, term, err := ReuseSingle(p, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("start postgres container, err: %s", err)
	}

	return db
}

func ReuseSingle(p *Reuse, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	p.mu.Lock()

	if p.db != nil {
		db = p.db
		term = func() {
			defer p.mu.Unlock()

			err := migrations.Down(db)
			if err != nil {
				log.Printf("failed down migraitions, %s", err)
			}
		}

		return db, term, nil
	}

	db, term, err = Run(migrations, initialQueries...)
	p.containerTerm = term
	term = func() {
		defer p.mu.Unlock()

		err := migrations.Down(db)
		if err != nil {
			log.Printf("failed down migraitions, %s", err)
		}
	}

	if err != nil {
		return db, term, err
	}

	p.db = db

	return db, term, err
}
