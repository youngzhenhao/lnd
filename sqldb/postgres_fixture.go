//go:build !js && !(windows && (arm || 386)) && !(linux && (ppc64 || mips || mipsle || mips64)) && !(netbsd || openbsd)

package sqldb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq" // Import the postgres driver.
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

const (
	testPgUser   = "test"
	testPgPass   = "test"
	testPgDBName = "test"
	PostgresTag  = "11"
)

// TestPgFixture is a test fixture that starts a Postgres 11 instance in a
// docker container.
type TestPgFixture struct {
	db       *sql.DB
	pool     *dockertest.Pool
	resource *dockertest.Resource
	host     string
	port     int
}

// NewTestPgFixture constructs a new TestPgFixture starting up a docker
// container running Postgres 11. The started container will expire in after
// the passed duration.
func NewTestPgFixture(t *testing.T, expiry time.Duration) *TestPgFixture {
	// Use a sensible default on Windows (tcp/http) and linux/osx (socket)
	// by specifying an empty endpoint.
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "Could not connect to docker")

	// Pulls an image, creates a container based on it and runs it.
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        PostgresTag,
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%v", testPgUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%v", testPgPass),
			fmt.Sprintf("POSTGRES_DB=%v", testPgDBName),
			"listen_addresses='*'",
		},
		Cmd: []string{
			"postgres",
			"-c", "log_statement=all",
			"-c", "log_destination=stderr",
			"-c", "max_connections=1000",
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that stopped container goes away
		// by itself.
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err, "Could not start resource")

	hostAndPort := resource.GetHostPort("5432/tcp")
	parts := strings.Split(hostAndPort, ":")
	host := parts[0]
	port, err := strconv.ParseInt(parts[1], 10, 64)
	require.NoError(t, err)

	fixture := &TestPgFixture{
		host: host,
		port: int(port),
	}
	databaseURL := fixture.GetConfig(testPgDBName).Dsn
	log.Infof("Connecting to Postgres fixture: %v\n", databaseURL)

	// Tell docker to hard kill the container in "expiry" seconds.
	require.NoError(t, resource.Expire(uint(expiry.Seconds())))

	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	pool.MaxWait = 120 * time.Second

	var testDB *sql.DB
	err = pool.Retry(func() error {
		testDB, err = sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}

		return testDB.Ping()
	})
	require.NoError(t, err, "Could not connect to docker")

	// Now fill in the rest of the fixture.
	fixture.db = testDB
	fixture.pool = pool
	fixture.resource = resource

	return fixture
}

// GetConfig returns the full config of the Postgres node.
func (f *TestPgFixture) GetConfig(dbName string) *PostgresConfig {
	return &PostgresConfig{
		Dsn: fmt.Sprintf(
			"postgres://%v:%v@%v:%v/%v?sslmode=disable",
			testPgUser, testPgPass, f.host, f.port, dbName,
		),
	}
}

// TearDown stops the underlying docker container.
func (f *TestPgFixture) TearDown(t *testing.T) {
	err := f.pool.Purge(f.resource)
	require.NoError(t, err, "Could not purge resource")
}

// NewTestPostgresDB is a helper function that creates a Postgres database for
// testing using the given fixture.
func NewTestPostgresDB(t *testing.T, fixture *TestPgFixture) *PostgresStore {
	t.Helper()

	// Create random database name.
	randBytes := make([]byte, 8)
	_, err := rand.Read(randBytes)
	if err != nil {
		t.Fatal(err)
	}

	dbName := "test_" + hex.EncodeToString(randBytes)

	t.Logf("Creating new Postgres DB '%s' for testing", dbName)

	_, err = fixture.db.ExecContext(
		context.Background(), "CREATE DATABASE "+dbName,
	)
	if err != nil {
		t.Fatal(err)
	}

	cfg := fixture.GetConfig(dbName)
	store, err := NewPostgresStore(cfg)
	require.NoError(t, err)

	return store
}
