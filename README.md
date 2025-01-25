# Containers, package for integration testing

```go
func Test_XXX(t *testing.T) {
	db := postgrescontainer.RunForTesting(t)

	...// Your code for testing
}
```

***Available containers***

- [postgres](README#Postgres)
- redis
- minio



## Postgres

### Run container

Run new container funcs run new postgres container, run provided migrations and run initial queries repectively

1. Run for testing
```go
func Test_XXX(t *testing.T) {
	db := postgrescontainer.RunForTesting(t,
		postgrescontainer.GooseMigrations("./testdata/migrations"),
		"INSERT INTO users (name) VALUES ('amidgo')",
	)

	...// Your code for testing
}


// postgrescontainer.RunForTesting code
func RunForTesting(t *testing.T, migrations Migrations, initialQueries ...string) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := RunContext(ctx, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("start postgres container, err: %s", err)
	}

	return db
}
```

2. Run new container via TestMain, not recommended

```go
// global variable
var db *sql.DB

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	containerDB, term, err := postgrescontainer.RunContext(
		ctx,
		postgrescontainer.GooseMigrations("./testdata/migrations"),
		"INSERT INTO users (name) VALUES ('amidgo')",
	)
	// The term func must be call regardless of function call
	defer term()

	// don't use log.Fatal funcs, use panic, otherwise defer funcs won't be called
	if err != nil {
		panic("failed run container, " + err.Error())
	}

	db = containerDB

	code := m.Run()
	os.Exit(code)
}
```

### Migrations

```go
type Migrations interface {
	UpContext(ctx context.Context, db *sql.DB) error
}
```

Migrations used to apply migrations for your container

***postgrescontainer*** package provided implementations:

1. GooseMigrations

```go
type gooseMigrations struct {
	folder string
}

func GooseMigrations(folder string) Migrations {
	return gooseMigrations{
		folder: folder,
	}
}

func (g gooseMigrations) UpContext(ctx context.Context, db *sql.DB) error {
	return goose.UpContext(ctx, db, g.folder)
}
```

2. EmptyMigrations

```go
var EmptyMigrations Migrations = emptyMigrations{}

type emptyMigrations struct{}

func (emptyMigrations) UpContext(context.Context, *sql.DB) error {
	return nil
}
```

### Reuse container

You can reuse single container over all project tests with Reuseable

1. Reuse for testing

```go
func Test_XXX(t *testing.T) {
	db := postgrescontainer.ReuseForTesting(t,
		postgrescontainer.GlobalReusable(),
		postgrescontainer.GooseMigrations("./testdata/migrations"),
		"INSERT INTO users (name) VALUES ('amidgo')",
	)

	...// Your code for testing
}

func ReuseForTesting(t *testing.T, reuse *Reusable, migrations Migrations, initialQueries ...string) *sql.DB {
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
```

2. Reuse container via TestMain, not recommended

```go
// global variable
var db *sql.DB

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	containerDB, term, err := postgrescontainer.ReuseForTesting(t,
		postgrescontainer.GlobalReusable(),
		postgrescontainer.GooseMigrations("./testdata/migrations"),
		"INSERT INTO users (name) VALUES ('amidgo')",
	)
	// The term func must be call regardless of function call
	defer term()

	// don't use log.Fatal funcs, use panic, otherwise defer funcs won't be called
	if err != nil {
		panic("failed reuse container, " + err.Error())
	}

	db = containerDB

	code := m.Run()
	os.Exit(code)
}
```

### Reusable

postgrescontainer.Reusable is a component for reuse containers, package provides two global Reusable:

1. GlobalReusable - create container at runtime, use RunContainer ***ccf***
2. GlobalEnvReusable - use external database from environment variables, use EnvContainer ***ccf***
3. You can create your own reusable

```go
type ReusableOption func(r *Reusable)
type CreateContainerFunc func(ctx context.Context) (postgresContainer, error)

func NewReusable(ccf CreateContainerFunc, opts ...ReusableOption) *Reusable
```

When ReuseContext is called, if the container was terminated or it's first ReuseContext call, reusable call ***ccf*** function to create new container

Every new reuse *sql.DB will use it's own schema with name public + inremented digit, when you call ReuseContext at first time, schema name will be public1

After calling term function, if nobody reuse container in this moment, start a timer after which terminate container

If call ReuseContext after timer starts, timer will be interrupted

Timer duration can be changed with ***WithWaitDuration***

```go
type ReusableOption func(r *Reusable)

func WithWaitDuration(duration time.Duration) ReusableOption {
	return func(r *Reusable) {
		r.waitDuration = duration
	}
}
```
