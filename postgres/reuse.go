package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amidgo/containers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultDuration = time.Second
)

var (
	globalReusable = Reuseable{
		ccf:          RunContainer,
		waitDuration: defaultDuration,
	}

	globalEnvReusable = Reuseable{
		ccf:          EnvContainer,
		waitDuration: defaultDuration,
	}
)

func GlobalReusable() *Reuseable {
	return &globalReusable
}

func GlobalEnvReusable() *Reuseable {
	return &globalEnvReusable
}

type ReuseableOption func(r *Reuseable)

var WithWaitDuration = func(duration time.Duration) ReuseableOption {
	return func(r *Reuseable) {
		r.waitDuration = duration
	}
}

func NewReusable(ccf CreateContainerFunc, opts ...ReuseableOption) *Reuseable {
	r := &Reuseable{
		ccf:          ccf,
		waitDuration: defaultDuration,
	}

	for _, op := range opts {
		op(r)
	}

	return r
}

type Reuseable struct {
	runDaemonOnce sync.Once
	ccf           CreateContainerFunc
	schemaCounter atomic.Int64
	dm            *containers.ReuseDaemon

	waitDuration time.Duration
}

func (r *Reuseable) runDaemon() {
	ccf := func(ctx context.Context) (any, error) {
		return r.ccf(ctx)
	}

	r.dm = containers.NewReuseDaemon(r.waitDuration, ccf)

	go r.dm.Run(context.Background())
}

func (r *Reuseable) runContext(ctx context.Context, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	r.runDaemonOnce.Do(r.runDaemon)

	pgCnt, err := r.enter(ctx)
	if err != nil {
		return nil, func() {}, fmt.Errorf("enter to reuse container, %w", err)
	}

	db, term, err = r.reuse(ctx, pgCnt, migrations, initialQueries...)
	if err != nil {
		return db, term, fmt.Errorf("reuse container, %w", err)
	}

	return db, term, nil
}

func (r *Reuseable) reuse(ctx context.Context, pgCnt postgresContainer, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	term = r.dm.Exit

	schemaName, err := r.createNewSchemaInContainer(ctx, pgCnt)
	if err != nil {
		return nil, term, err
	}

	db, err = connectToSchema(ctx, pgCnt, schemaName)
	if err != nil {
		return db, term, err
	}

	term = func() {
		_ = db.Close()
		r.dm.Exit()
	}

	err = migrations.UpContext(ctx, db)
	if err != nil {
		return db, term, fmt.Errorf("up migrations, %w", err)
	}

	for _, initialQuery := range initialQueries {
		_, execErr := db.ExecContext(ctx, initialQuery)
		if execErr != nil {
			return db, term, fmt.Errorf("exec %s query, %w", initialQuery, execErr)
		}
	}

	return db, term, nil
}

func (r *Reuseable) createNewSchemaInContainer(ctx context.Context, pgCnt postgresContainer) (schemaName string, err error) {
	connString, err := pgCnt.ConnectionString(ctx, "sslmode=disable")
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

	schemaName = fmt.Sprintf("public%d", schemaCount)

	query := fmt.Sprintf("CREATE SCHEMA %s", schemaName)

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("create schema %s, %w", schemaName, err)
	}

	return schemaName, nil
}

func connectToSchema(ctx context.Context, pgCnt postgresContainer, schemaName string) (*sql.DB, error) {
	connString, err := pgCnt.ConnectionString(ctx, "sslmode=disable", "search_path="+schemaName)
	if err != nil {
		return nil, fmt.Errorf("get connection string to specific schema, schema_name=%s, %w", schemaName, err)
	}

	conn, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("open connection, %w", err)
	}

	db := stdlib.OpenDBFromPool(conn)

	return db, nil
}

func (r *Reuseable) enter(ctx context.Context) (postgresContainer, error) {
	cnt, err := r.dm.Enter(ctx)
	if err != nil {
		return nil, err
	}

	return cnt.(postgresContainer), nil
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

func EnvContainer(ctx context.Context) (postgresContainer, error) {
	return nil, nil
}
