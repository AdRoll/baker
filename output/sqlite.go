package output

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/AdRoll/baker"
)

// SQLiteDesc declares the standard (non-raw) SQLite output.
var SQLiteDesc = baker.OutputDesc{
	Name:   "SQLite",
	New:    newSQLiteWriter(false),
	Config: &SQLiteWriterConfig{},
	Help:   "Writes a chosen set of fields as table columns into a local SQLite database file",
	Raw:    false,
}

// SQLiteRawDesc declares the raw SQLite output.
var SQLiteRawDesc = baker.OutputDesc{
	Name:   "SQLiteRaw",
	New:    newSQLiteWriter(true),
	Config: &SQLiteRawWriterConfig{},
	Help:   "Writes a chosen set of fields, plus the raw record, as table columns into a local SQLite database file",
	Raw:    true,
}

// SQLiteWriterConfig holds the configuration parameters for the
// standard SQLite baker output.
type SQLiteWriterConfig struct {
	PathString string   `help:"Path to local SQLite file to write the results to. Will be created if it does not exist. Can contain {{.ShardId}} and {{.Field}} for replacement" required:"true"`
	TableName  string   `help:"Table name to which to write the records to." required:"true"`
	PreRun     []string `help:"List of SQL statements to run at startup (before table creation)."`
	PostRun    []string `help:"List of SQL statements to run at exit. (good place to create indexes if needed)."`
	Clear      bool     `help:"Whether DELETE should be run on TableName before starting. By default, if the target file already exists and has a table, this output will append to that table. This flag, if set, makes it so that the table is truncated first."`
	Vacuum     bool     `help:"Should we run VACUUM at the end? Note that PostRun can't take VACUUM commands because it's run inside a transaction. If you want to vacuum, pass this argument. (Useful if your PostRun command deletes lots of data, and you want to shrink the file size)."`
	Wal        bool     `help:"Send PRAGMA journal_mode=wal; before starting. This turns on write-ahead logging for the SQLite file. This mode is usually more friendly to bulk I/O operations."`
	PageSize   int64    `help:"The page size to use for SQLite. By default, we use whatever SQLite decides to use as default."`
}

// convert to raw config, which is a superset, so that the sqlite output can
// always use one type only
func (cfg *SQLiteWriterConfig) convert() *SQLiteRawWriterConfig {
	return &SQLiteRawWriterConfig{
		PathString: cfg.PathString,
		TableName:  cfg.TableName,
		PreRun:     cfg.PreRun,
		PostRun:    cfg.PostRun,
		Clear:      cfg.Clear,
		Vacuum:     cfg.Vacuum,
		Wal:        cfg.Wal,
		PageSize:   cfg.PageSize,
	}
}

// SQLiteRawWriterConfig holds the configuration parameters for the
// raw SQLite baker output.
type SQLiteRawWriterConfig struct {
	PathString     string   `help:"Path to local SQLite file to write the results to. Will be created if it does not exist. Can contain {{.ShardId}} and {{.Field}} for replacement" required:"true"`
	TableName      string   `help:"Table name to which to write the records to." required:"true"`
	PreRun         []string `help:"List of SQL statements to run at startup (before table creation)."`
	PostRun        []string `help:"List of SQL statements to run at exit. (good place to create indexes if needed)."`
	Clear          bool     `help:"Whether DELETE should be run on TableName before starting. By default, if the target file already exists and has a table, this output will append to that table. This flag, if set, makes it so that the table is truncated first."`
	Vacuum         bool     `help:"Should we run VACUUM at the end? Note that PostRun can't take VACUUM commands because it's run inside a transaction. If you want to vacuum, pass this argument. (Useful if your PostRun command deletes lots of data, and you want to shrink the file size)."`
	Wal            bool     `help:"Send PRAGMA journal_mode=wal; before starting. This turns on write-ahead logging for the SQLite file. This mode is usually more friendly to bulk I/O operations."`
	PageSize       int64    `help:"The page size to use for SQLite. By default, we use whatever SQLite decides to use as default."`
	RecordBlobName string   `help:"Name of the column in which the whole raw record should be put." required:"true"`
}

type SQLiteWriter struct {
	cfg        *SQLiteRawWriterConfig
	pathString string
	fieldNames []string
	nEvents    int64
	isRaw      bool    // are we a raw sqlite writer?
	tx         *sql.Tx // main transaction
	conn       *sql.DB
}

func renderSQLitePathString(pathString string, shardID int, field string) (string, error) {
	// This function substitutes {{ShardId}} and {{Field}} in PathString.
	var templ *template.Template
	var doc bytes.Buffer
	replacementVars := map[string]string{
		"ShardId": fmt.Sprintf("%04d", shardID),
		"Field":   field,
	}

	templ, err := template.New("sqlite").Parse(pathString)
	if err != nil {
		return "", err
	}

	err = templ.Execute(&doc, replacementVars)
	if err != nil {
		return "", err
	}
	dir := doc.String()
	if dir == "" {
		return "", fmt.Errorf("empty rendered path template")
	}
	return dir, nil
}

// runSQLCommands is an helper function that will run some commands on an SQL transaction.
func runSQLCommands(tx *sql.Tx, commands []string) error {
	for _, command := range commands {
		prep, err := tx.Prepare(command)
		if err != nil {
			return err
		}
		if _, err := prep.Exec(); err != nil {
			prep.Close()
			return err
		}
		prep.Close()
	}
	return nil
}

func newSQLiteWriter(isRaw bool) func(baker.OutputParams) (baker.Output, error) {
	return func(cfg baker.OutputParams) (baker.Output, error) {
		if cfg.DecodedConfig == nil {
			return nil, fmt.Errorf("no config provided")
		}

		// Convert and validate configuration
		var dcfg *SQLiteRawWriterConfig
		if isRaw {
			dcfg = cfg.DecodedConfig.(*SQLiteRawWriterConfig)
		} else {
			dcfg = cfg.DecodedConfig.(*SQLiteWriterConfig).convert()
		}

		path, err := renderSQLitePathString(dcfg.PathString, cfg.Index, "")
		if err != nil {
			return nil, fmt.Errorf("can't render PathString: %s", err)
		}

		// Make a record of all field names; they will be needed when we create
		// an SQL table in the SQLite file.
		var fieldNames []string
		for _, fidx := range cfg.Fields {
			fieldNames = append(fieldNames, cfg.FieldName(fidx))
		}

		sqlw := &SQLiteWriter{
			cfg:        dcfg,
			pathString: path,
			fieldNames: fieldNames,
			isRaw:      isRaw,
		}

		if err = sqlw.setup(); err != nil {
			return nil, fmt.Errorf("setup error: %v", err)
		}

		return sqlw, nil
	}
}

func (c *SQLiteWriter) setup() error {
	var err error

	defer func() {
		if err != nil && c.conn != nil {
			c.conn.Close()
		}
	}()

	if err = c.vetIdentifiers(); err != nil {
		return err
	}

	// Set up database handle
	c.conn, err = sql.Open("sqlite3", c.pathString)
	if err != nil {
		return fmt.Errorf("sql.Open: %s", err)
	}

	if err = c.setDBSettings(c.conn); err != nil {
		return fmt.Errorf("set global settings: %s", err)
	}

	// Open transaction. The entire thing will run in just this one
	// transaction.
	c.tx, err = c.conn.Begin()
	if err != nil {
		return fmt.Errorf("Cannot start transaction: %s", err)
	}

	// Run the pre-run commands
	if err = runSQLCommands(c.tx, c.cfg.PreRun); err != nil {
		c.tx.Rollback()
		return fmt.Errorf("Cannot run pre-SQL commands: %s", err)
	}

	return nil
}

// isPrintable reports whether a string contains only printable runes.
func isPrintable(str string) bool {
	const firstPrintable = 32 // ASCII space

	for _, r := range str {
		if r < firstPrintable || r > unicode.MaxASCII {
			return false
		}
	}

	return true
}

// vetIdentifiers checks all identifiers respect some rule so that we can use
// them safely in queries.
func (c *SQLiteWriter) vetIdentifiers() error {
	ids := map[string]string{
		"TableName":      c.cfg.TableName,
		"RecordBlobName": c.cfg.RecordBlobName,
	}

	for _, f := range c.fieldNames {
		ids["field "+f] = f
	}

	for name, id := range ids {
		if !isPrintable(id) {
			return fmt.Errorf("%s contains non-printable characters", name)
		}
	}

	return nil
}

func (c *SQLiteWriter) Run(input <-chan baker.OutputRecord, upch chan<- string) error {
	err := c.doRun(input)
	if err != nil {
		return fmt.Errorf("SQLite writer failed: %v", err)
	}

	// Send the file to upch
	if pathname, err := filepath.Abs(c.pathString); err != nil {
		return fmt.Errorf("taking absolute path failed: %v", err)
	} else {
		upch <- pathname
	}

	return nil
}

// sqliteQuote returns a manually escaped string replacing single quotes with double-single quotes
func sqliteQuote(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "''") + "'"
}

// prepInsertStatement prepares and returns the statement inserting records.
func (c *SQLiteWriter) prepInsertStatement(tx *sql.Tx) (*sql.Stmt, error) {
	var qmarks []string
	for range c.fieldNames {
		qmarks = append(qmarks, "?")
	}
	if c.isRaw {
		qmarks = append(qmarks, "?")
	}

	stmt := fmt.Sprintf("INSERT INTO %s VALUES(%s)", sqliteQuote(c.cfg.TableName), strings.Join(qmarks, ","))
	return tx.Prepare(stmt)
}

func (c *SQLiteWriter) setDBSettings(conn *sql.DB) error {
	if c.cfg.PageSize > 0 {
		if _, err := conn.Exec(fmt.Sprintf("PRAGMA page_size=%d", c.cfg.PageSize)); err != nil {
			return fmt.Errorf("PRAGMA page_size=%d failed: %s", c.cfg.PageSize, err)
		}
	}

	if c.cfg.Wal {
		if _, err := conn.Exec("PRAGMA journal_mode=wal"); err != nil {
			return fmt.Errorf("PRAGMA journal_mode=wal failed: %s", err)
		}
	}

	return nil
}

// maybeTruncate truncates the table (if configured to do so)
func (c *SQLiteWriter) maybeTruncate(tx *sql.Tx) error {
	if c.cfg.Clear {
		stmt, err := tx.Prepare("DELETE FROM " + sqliteQuote(c.cfg.TableName))
		if err != nil {
			return fmt.Errorf("prepare truncate statement: %s", err)
		}
		if _, err = stmt.Exec(); err != nil {
			stmt.Close()
			return fmt.Errorf("truncate table: %s", err)
		}
		stmt.Close()
	}

	return nil
}

// setupTable either creates the table or, in case it already exists and the
// config has Clear=true, we truncate the table.
func (c *SQLiteWriter) setupTable(tx *sql.Tx) error {
	// Build the SQL statement that creates the table.
	// CREATE TABLE IF NOT EXISTS ? ( ?, ?, ... )
	var fields []string
	for _, f := range c.fieldNames {
		fields = append(fields, sqliteQuote(f))
	}
	if c.isRaw {
		// Raw record is a BLOB column.
		fields = append(fields, sqliteQuote(c.cfg.RecordBlobName)+" BLOB")
	}

	sstmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)",
		sqliteQuote(c.cfg.TableName), strings.Join(fields, ","))

	stmt, err := tx.Prepare(sstmt)
	if err != nil {
		return fmt.Errorf("prepare create statement: %s", err)
	}
	if _, err = stmt.Exec(); err != nil {
		stmt.Close()
		return fmt.Errorf("create table: %s", err)
	}
	stmt.Close()

	return c.maybeTruncate(tx)
}

func (c *SQLiteWriter) doRun(input <-chan baker.OutputRecord) error {
	// Install a deferred rollback; if something errors out, we execute
	// tx.Rollback().
	//
	// At the end we set commitDone to true so this logic won't actually
	// run if commit is successful.
	commitDone := false
	defer func() {
		if !commitDone {
			c.tx.Rollback()
		}
	}()

	if err := c.setupTable(c.tx); err != nil {
		return fmt.Errorf("setup table: %s", err)
	}
	insert, err := c.prepInsertStatement(c.tx)
	if err != nil {
		return fmt.Errorf("build insert statement: %s", err)
	}

	ncols := len(c.fieldNames)
	if c.isRaw {
		ncols++
	}

	values := make([]interface{}, ncols)

	for lldata := range input {
		for i, str := range lldata.Fields {
			values[i] = str
		}
		if c.isRaw {
			values[len(values)-1] = lldata.Record
		}
		_, err = insert.Exec(values...)
		if err != nil {
			insert.Close()
			return fmt.Errorf("cannot insert to SQLite file: %s", err)
		}
		c.nEvents++
	}
	insert.Close()

	// Run final post-commands, if any are configured.
	if err = runSQLCommands(c.tx, c.cfg.PostRun); err != nil {
		return fmt.Errorf("cannot run post commands: %s", err)
	}

	// Commit all changes.
	if err = c.tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit SQLite transaction: %s", err)
	}
	commitDone = true

	// Vacuum the SQLite file, if defined so in configuration
	if c.cfg.Vacuum {
		_, err = c.conn.Exec("VACUUM")
		if err != nil {
			return fmt.Errorf("cannot VACUUM SQLite file: %s", err)
		}
	}

	return nil
}

func (c *SQLiteWriter) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: c.nEvents,
	}
}

func (c *SQLiteWriter) CanShard() bool {
	// github.com/mattn/go-sqlite3 supports concurrency only for reading,
	// so the output isn't concurrent-safe and it supports sharding
	// only with different file names
	return strings.Contains(c.cfg.PathString, "{{.ShardId}}")
}

func (c *SQLiteWriter) SupportConcurrency() bool {
	// github.com/mattn/go-sqlite3 supports concurrency only for reading,
	// so the output isn't concurrent-safe and it supports "concurrency"
	// only with different file names
	return strings.Contains(c.cfg.PathString, "{{.ShardId}}")
}
