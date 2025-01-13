package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/amidgo/containers"
)

var (
	globalReusable = Reuseable{
		ccf: runContainer,
	}

	globalEnvReusable = Reuseable{
		ccf: envContainer,
	}
)

func GlobalReusable() *Reuseable {
	return &globalReusable
}

func GlobalEnvReusable() *Reuseable {
	return &globalEnvReusable
}

type Reuseable struct {
	mu            sync.Mutex
	cnt           postgresContainer
	ccf           createConatainerFunc
	schemaCounter atomic.Int64
	aliveUsers    atomic.Int64
}

func (r *Reuseable) runContext(ctx context.Context, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	err = r.init(ctx)
	if err != nil {
		return nil, func() {}, fmt.Errorf("init reusable, %w", err)
	}

	if r.cnt == nil {
		panic("nil reusable container value after initialize")
	}

	db, term, err = r.reuse(ctx, migrations)
	if err != nil {
		return db, term, fmt.Errorf("reuse container, %w", err)
	}

	return db, term, nil
}

func (r *Reuseable) reuse(ctx context.Context, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	r.incrAliveUsers()

	term = r.decrAliveUsers

	schemaName, err := r.createNewSchemaInContainer(ctx)
	if err != nil {
		return nil, term, err
	}

	connString, err := r.cnt.ConnectionString(ctx, "sslmode=disable search_path="+schemaName)
	if err != nil {
		return nil, term, fmt.Errorf("get connection string to specific schema, schema_name=%s, %w", schemaName, err)
	}

	db, err = sql.Open("pgx", connString)
	if err != nil {
		return nil, term, fmt.Errorf("open connection to specific schema, schema_name=%s, %w", schemaName, err)
	}

	err = migrations.Up(db)
	if err != nil {
		return db, term, err
	}

	for _, initialQuery := range initialQueries {
		_, execErr := db.ExecContext(ctx, initialQuery)
		if execErr != nil {
			return db, term, fmt.Errorf("exec %s query, %w", initialQuery, execErr)
		}
	}

	return db, term, nil
}

func (r *Reuseable) createNewSchemaInContainer(ctx context.Context) (schemaName string, err error) {
	connString, err := r.cnt.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", fmt.Errorf("get connection string, %w", err)
	}

	baseDB, err := sql.Open("pgx", connString)
	if err != nil {
		return "", fmt.Errorf("open connection to db, %w", err)
	}

	defer baseDB.Close()

	schemaName, err = r.createSchema(ctx, baseDB)
	if err != nil {
		return "", err
	}

	return schemaName, nil
}

func (r *Reuseable) createSchema(ctx context.Context, db *sql.DB) (schemaName string, err error) {
	schemaCount := r.schemaCounter.Add(1)

	const baseSchemaName = "public"

	schemaName = baseSchemaName + strconv.FormatInt(schemaCount, 10)

	_, err = db.ExecContext(ctx, "CREATE SCHEMA $1", schemaName)
	if err != nil {
		return "", fmt.Errorf("create schema %s, %w", schemaName, err)
	}

	return schemaName, nil
}

func (r *Reuseable) incrAliveUsers() {
	r.aliveUsers.Add(1)
}

func (r *Reuseable) decrAliveUsers() {
	newValue := r.aliveUsers.Add(-1)

	switch {
	case newValue < 0:
		panic("unexpected negative integer, r.aliveUsers = " + strconv.FormatInt(newValue, 10))
	case newValue == 0:
		r.reset()
	}
}

func (r *Reuseable) init(ctx context.Context) error {
	if r.cnt != nil {
		return nil
	}

	cnt, err := r.ccf(ctx)
	if err != nil {
		return fmt.Errorf("run create container function r.ccf, %w", err)
	}

	r.cnt = cnt

	return nil
}

func (r *Reuseable) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.cnt.Terminate(context.Background())
	if err != nil {
		log.Printf("terminating postgres container, %s", err)
	}

	r.cnt = nil
}

func ReuseForTesting(t *testing.T, reuse *Reuseable, migrations Migrations, initialQueries ...string) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := ReuseContext(ctx, reuse, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("reuse container, err: %s", err)
	}

	return db
}

func Reuse(reuse *Reuseable, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	return ReuseContext(context.Background(), reuse, migrations, initialQueries...)
}

func ReuseContext(ctx context.Context, reuse *Reuseable, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	return reuse.runContext(ctx, migrations, initialQueries...)
}

func envContainer(ctx context.Context) (postgresContainer, error) {
	return nil, nil
}
