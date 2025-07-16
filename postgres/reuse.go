package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/amidgo/containers"
	"github.com/amidgo/containers/postgres/migrations"
)

func ReuseForTesting(
	t *testing.T,
	reuse *Reusable,
	mig migrations.Migrations,
	initialQueries ...migrations.Query,
) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := Reuse(ctx, reuse, mig, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("reuse container, err: %s", err)

		return nil
	}

	return db
}

func Reuse(
	ctx context.Context,
	reuse *Reusable,
	mig migrations.Migrations,
	initialQueries ...migrations.Query,
) (db *sql.DB, term func(), err error) {
	return reuse.run(ctx, mig, initialQueries...)
}

const defaultDuration = time.Second

type ReusableOption func(r *Reusable)

func WithWaitDuration(duration time.Duration) ReusableOption {
	return func(r *Reusable) {
		r.daemonWaitDuration = duration
	}
}

type Reusable struct {
	ccf CreateContainerFunc

	runDaemonOnce      sync.Once
	dm                 *containers.ReusableDaemon
	stopDaemon         context.CancelFunc
	daemonWaitDuration time.Duration
}

func NewReusable(ccf CreateContainerFunc, opts ...ReusableOption) *Reusable {
	r := &Reusable{
		ccf:                ccf,
		daemonWaitDuration: defaultDuration,
	}

	for _, op := range opts {
		op(r)
	}

	return r
}

func (r *Reusable) runDaemon() {
	ccf := func(ctx context.Context) (any, error) {
		return r.ccf(ctx)
	}

	ctx, cancel := context.WithCancel(context.Background())

	daemon := containers.RunReusableDaemon(ctx, r.daemonWaitDuration, ccf)

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
	mig migrations.Migrations,
	initialQueries ...migrations.Query,
) (db *sql.DB, term func(), err error) {
	r.runDaemonOnce.Do(r.runDaemon)

	pgCnt, err := r.enter(ctx)
	if err != nil {
		return nil, func() {}, fmt.Errorf("enter to reuse container, %w", err)
	}

	db, term, err = r.reuse(ctx, pgCnt, mig, initialQueries...)
	if err != nil {
		return db, term, fmt.Errorf("reuse container, %w", err)
	}

	return db, term, nil
}

func (r *Reusable) reuse(
	ctx context.Context,
	pgCnt Container,
	mig migrations.Migrations,
	initialQueries ...migrations.Query,
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

	if mig != nil {
		err = mig.Up(ctx, db)
		if err != nil {
			return db, term, fmt.Errorf("up migrations, %w", err)
		}
	}

	for _, initialQuery := range initialQueries {
		err = migrations.ExecQuery(ctx, db, initialQuery)
		if err != nil {
			return db, term, err
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
	schemaName = fmt.Sprintf("public%d", rand.Int64())

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
