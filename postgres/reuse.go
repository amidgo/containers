package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		recvCh:       make(chan reuseContainerResponse),
		sendCh:       make(chan reuseContainerRequest),
	}

	globalEnvReusable = Reuseable{
		ccf:          EnvContainer,
		waitDuration: defaultDuration,
		recvCh:       make(chan reuseContainerResponse),
		sendCh:       make(chan reuseContainerRequest),
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
		recvCh:       make(chan reuseContainerResponse),
		sendCh:       make(chan reuseContainerRequest),
	}

	for _, op := range opts {
		op(r)
	}

	return r
}

type reuseCommand uint8

const (
	reuseCommandEnter reuseCommand = iota
	reuseCommandExit
)

type reuseContainerRequest struct {
	reuseCmd reuseCommand
	ctx      context.Context
}

type invalidReuseCommandError struct {
	cmd reuseCommand
}

func (i invalidReuseCommandError) Error() string {
	return fmt.Sprintf("invalid reuse command: %d", i.cmd)
}

type reuseContainerResponse struct {
	pgCnt postgresContainer
	err   error
}

type Reuseable struct {
	runDaemonOnce sync.Once
	ccf           CreateContainerFunc
	schemaCounter atomic.Int64

	sendCh chan reuseContainerRequest
	recvCh chan reuseContainerResponse

	waitDuration time.Duration
}

type daemon struct {
	activeUsers  int
	pgCnt        postgresContainer
	waitDuration time.Duration

	recvCh chan reuseContainerRequest
	sendCh chan reuseContainerResponse

	ccf CreateContainerFunc
}

func (d *daemon) run() {
	for req := range d.recvCh {
		d.handleReuseCommand(req.ctx, req.reuseCmd)
	}
}

func (d *daemon) handleReuseCommand(ctx context.Context, reuseCmd reuseCommand) {
	switch reuseCmd {
	case reuseCommandEnter:
		d.activeUsers++
	case reuseCommandExit:
		d.activeUsers--
	default:
		d.sendCh <- reuseContainerResponse{
			err: invalidReuseCommandError{cmd: reuseCmd},
		}

		return
	}

	switch {
	case d.activeUsers > 0:
		d.handlePositiveActiveUsers(ctx)
	case d.activeUsers == 0:
		d.handleZeroActiveUsers(ctx)
	case d.activeUsers <= 0:
		panic("reuse container term func called twice, negative amount of active users")
	}
}

func (d *daemon) handlePositiveActiveUsers(ctx context.Context) {
	if d.pgCnt == nil {
		pgCnt, err := d.ccf(ctx)
		if err != nil {
			d.sendCh <- reuseContainerResponse{
				err: fmt.Errorf("create new container, %w", err),
			}

			return
		}

		d.pgCnt = pgCnt
	}

	d.sendCh <- reuseContainerResponse{
		pgCnt: d.pgCnt,
	}
}

func (d *daemon) handleZeroActiveUsers(ctx context.Context) {
	select {
	case <-time.After(d.waitDuration):
		d.clearContainer(ctx)
		d.sendCh <- reuseContainerResponse{}
	case req := <-d.recvCh:
		switch req.reuseCmd {
		case reuseCommandEnter:
			d.activeUsers++
		case reuseCommandExit:
			panic("unexpected exit command in handleZeroActiveUsers")
		default:
			d.sendCh <- reuseContainerResponse{
				err: invalidReuseCommandError{cmd: req.reuseCmd},
			}
		}
	}
}

func (d *daemon) clearContainer(ctx context.Context) {
	err := d.pgCnt.Terminate(ctx)
	if err != nil {
		log.Printf("failed terminate container, %s", err)
	}

	d.pgCnt = nil
}

func (r *Reuseable) runDaemon() {
	daemon := daemon{
		waitDuration: r.waitDuration,
		recvCh:       r.sendCh,
		sendCh:       r.recvCh,
		ccf:          r.ccf,
	}

	go daemon.run()
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
	term = r.exit

	schemaName, err := r.createNewSchemaInContainer(ctx, pgCnt)
	if err != nil {
		return nil, term, err
	}

	db, err = connectToSchema(ctx, pgCnt, schemaName)
	if err != nil {
		return db, term, err
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
	r.sendCh <- reuseContainerRequest{
		ctx:      ctx,
		reuseCmd: reuseCommandEnter,
	}

	resp := <-r.recvCh

	return resp.pgCnt, resp.err
}

func (r *Reuseable) exit() {
	r.sendCh <- reuseContainerRequest{
		ctx:      context.Background(),
		reuseCmd: reuseCommandExit,
	}

	<-r.recvCh
}

func ReuseForTesting(t *testing.T, reuse *Reuseable, migrations Migrations, initialQueries ...string) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := ReuseContext(ctx, reuse, migrations, initialQueries...)
	t.Cleanup(closeDB(db))
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
