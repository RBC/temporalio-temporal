package test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
	"go.temporal.io/server/common/convert"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/tests/testutils"
)

// SetupSchemaTestBase is the base test suite for all tests
// that exercise schema setup using the schema tool
type SetupSchemaTestBase struct {
	suite.Suite
	*require.Assertions
	rand       *rand.Rand
	Logger     log.Logger
	DBName     string
	db         DB
	pluginName string
}

// SetupSuiteBase sets up the test suite
func (tb *SetupSchemaTestBase) SetupSuiteBase(db DB, pluginName string) {
	tb.Assertions = require.New(tb.T()) // Have to define our overridden assertions in the test setup. If we did it earlier, tb.T() will return nil
	tb.Logger = log.NewTestLogger()
	tb.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	tb.DBName = fmt.Sprintf("setup_test_%v", tb.rand.Int63())
	err := db.CreateDatabase(tb.DBName)
	if err != nil {
		tb.Logger.Fatal("error creating database, ", tag.Error(err))
	}
	tb.db = db
	tb.pluginName = pluginName
}

// TearDownSuiteBase tears down the test suite
func (tb *SetupSchemaTestBase) TearDownSuiteBase() {
	tb.NoError(tb.db.DropDatabase(tb.DBName))
	tb.db.Close()
}

// RunSetupTest exercises the SetupSchema task
func (tb *SetupSchemaTestBase) RunSetupTest(
	app *cli.App, db DB, dbNameFlag string, sqlFileContent string, expectedTables []string) {
	// test command fails without required arguments
	command := append(tb.getCommandBase(), []string{
		dbNameFlag, tb.DBName,
		"-q",
		"setup-schema",
	}...)
	tb.NoError(app.Run(command))
	tables, err := db.ListTables()
	tb.Nil(err)
	tb.Equal(0, len(tables))

	tmpDir := testutils.MkdirTemp(tb.T(), "", "setupSchemaTestDir")
	sqlFile := testutils.CreateTemp(tb.T(), tmpDir, "setupSchema.cliOptionsTest")

	_, err = sqlFile.WriteString(sqlFileContent)
	tb.NoError(err)

	// make sure command doesn't succeed without version or disable-version
	command = append(tb.getCommandBase(), []string{
		dbNameFlag, tb.DBName,
		"-q",
		"setup-schema",
		"-f", sqlFile.Name(),
	}...)
	tb.NoError(app.Run(command))
	tables, err = db.ListTables()
	tb.Nil(err)
	tb.Equal(0, len(tables))

	for i := 0; i < 4; i++ {

		ver := convert.Int32ToString(tb.rand.Int31())
		versioningEnabled := (i%2 == 0)

		// test overwrite with versioning works
		if versioningEnabled {
			command = append(tb.getCommandBase(), []string{
				dbNameFlag, tb.DBName,
				"-q",
				"setup-schema",
				"-f", sqlFile.Name(),
				"-version", ver,
				"-o",
			}...)
			tb.NoError(app.Run(command))
		} else {
			command = append(tb.getCommandBase(), []string{
				dbNameFlag, tb.DBName,
				"-q",
				"setup-schema",
				"-f", sqlFile.Name(),
				"-d",
				"-o",
			}...)
			tb.NoError(app.Run(command))
		}

		expectedTables := getExpectedTables(versioningEnabled, expectedTables)
		tables, err = db.ListTables()
		tb.Nil(err)
		tb.Equal(len(expectedTables), len(tables))

		for _, t := range tables {
			_, ok := expectedTables[t]
			tb.True(ok)
			delete(expectedTables, t)
		}
		tb.Equal(0, len(expectedTables))

		gotVer, err := db.ReadSchemaVersion()
		if versioningEnabled {
			tb.Nil(err)
			tb.Equal(ver, gotVer)
		} else {
			tb.NotNil(err)
		}
	}
}

func (tb *SetupSchemaTestBase) getCommandBase() []string {
	command := []string{"./tool"}
	if tb.pluginName != "" {
		command = append(command, []string{
			"-pl", tb.pluginName,
		}...)
	}
	return command
}

func getExpectedTables(versioningEnabled bool, wantTables []string) map[string]struct{} {
	expectedTables := make(map[string]struct{})
	for _, tab := range wantTables {
		expectedTables[tab] = struct{}{}
	}
	if versioningEnabled {
		expectedTables["schema_version"] = struct{}{}
		expectedTables["schema_update_history"] = struct{}{}
	}
	return expectedTables
}
