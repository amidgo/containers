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
	"github.com/amidgo/containers/postgres/migrations"
)

const defaultDuration = time.Second

type ReusableOption func(r *Reusable)

func WithWaitDuration(duration time.Duration) ReusableOption {
	return func(r *Reusable) {
		r.waitDuration = duration
	}
}

func NewReusable(creator Creator, opts ...ReusableOption) *Reusable {
	r := &Reusable{
		creator:      creator,
		waitDuration: defaultDuration,
	}

	for _, op := range opts {
		op(r)
	}

	return r
}

type Reusable struct {
	runDaemonOnce sync.Once
	creator       Creator
	schemaCounter atomic.Int64
	dm            *containers.ReusableDaemon
	stopDaemon    context.CancelFunc

	waitDuration time.Duration
}

func (r *Reusable) runDaemon() {
	ccf := func(ctx context.Context) (any, error) {
		return r.creator.Create(ctx)
	}

	ctx, cancel := context.WithCancel(context.Background())

	daemon := containers.RunReusableDaemon(ctx, r.waitDuration, ccf)

	r.dm = daemon
	r.stopDaemon = cancel
}

func (r *Reusable) Terminate(ctx context.Context) error {
	r.stopDaemon()

	select {
	case <-r.dm.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (r *Reusable) run(
	ctx context.Context,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
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

func (r *Reusable) reuse(
	ctx context.Context,
	pgCnt Container,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
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

	if migrations != nil {
		err = migrations.Up(ctx, db)
		if err != nil {
			return db, term, fmt.Errorf("up migrations, %w", err)
		}
	}

	for _, initialQuery := range initialQueries {
		_, execErr := db.ExecContext(ctx, initialQuery)
		if execErr != nil {
			return db, term, fmt.Errorf("exec %s query, %w", initialQuery, execErr)
		}
	}

	return db, term, nil
}

func (r *Reusable) createNewSchemaInContainer(ctx context.Context, pgCnt Container) (schemaName string, err error) {
	baseDB, err := pgCnt.Connect(ctx, "sslmode=disable")
	if err != nil {
		return "", fmt.Errorf("connect to database, %w", err)
	}

	defer baseDB.Close()

	schemaName, err = r.createSchema(ctx, baseDB)
	if err != nil {
		return "", err
	}

	return schemaName, nil
}

func (r *Reusable) createSchema(ctx context.Context, db *sql.DB) (schemaName string, err error) {
	schemaCount := r.schemaCounter.Add(1)

	schemaName = fmt.Sprintf("public%d", schemaCount)

	query := fmt.Sprintf("CREATE SCHEMA %s", schemaName)

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("create schema %s, %w", schemaName, err)
	}

	return schemaName, nil
}

func connectToSchema(ctx context.Context, pgCnt Container, schemaName string) (*sql.DB, error) {
	db, err := pgCnt.Connect(ctx, "sslmode=disable", "search_path="+schemaName)
	if err != nil {
		return nil, fmt.Errorf("connect to databse, schema_name=%s, %w", schemaName, err)
	}

	return db, nil
}

func (r *Reusable) enter(ctx context.Context) (Container, error) {
	cnt, err := r.dm.Enter(ctx)
	if err != nil {
		return nil, err
	}

	return cnt.(Container), nil
}

func ReuseForTesting(
	t *testing.T,
	reuse *Reusable,
	migrations migrations.Migrations,
	initialQueries ...string,
) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := Reuse(ctx, reuse, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("reuse container, err: %s", err)
	}

	return db
}

func Reuse(
	ctx context.Context,
	reuse *Reusable,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
	return reuse.run(ctx, migrations, initialQueries...)
}

func EnvContainer(ctx context.Context) (Container, error) {
	return nil, nil
}
