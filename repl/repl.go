package repl

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"github.com/arijiiiitttt/dinoDB/engine"
	"github.com/arijiiiitttt/dinoDB/query"
)

type REPL struct {
	db *engine.DB
	tx *engine.Transaction 
}

func New(db *engine.DB) *REPL {
	return &REPL{db: db}
}

func (r *REPL) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("GoDB Interactive Shell")
	fmt.Println("Type SQL commands and press Enter.")
	fmt.Println("Type .help for available commands, .exit to quit.")
	fmt.Println(strings.Repeat("─", 52))

	for {
		if r.tx != nil {
			fmt.Print("godb (tx)> ")
		} else {
			fmt.Print("godb> ")
		}

		if !scanner.Scan() {
			break 
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, ".") {
			if r.handleDotCommand(line) {
				return
			}
			continue
		}

		r.execute(line)
	}

	fmt.Println("\nBye!")
}

func (r *REPL) execute(input string) {
	upper := strings.ToUpper(strings.TrimSpace(input))

	switch upper {
	case "BEGIN":
		if r.tx != nil {
			printError("already inside a transaction — COMMIT or ROLLBACK first")
			return
		}
		r.tx = r.db.Begin()
		printStatus("ok", "transaction started")
		return

	case "COMMIT":
		if r.tx == nil {
			printError("no active transaction")
			return
		}
		if err := r.tx.Commit(); err != nil {
			printError(err.Error())
		} else {
			printStatus("committed", "transaction committed successfully")
		}
		r.tx = nil
		return

	case "ROLLBACK":
		if r.tx == nil {
			printError("no active transaction")
			return
		}
		if err := r.tx.Rollback(); err != nil {
			printError(err.Error())
		} else {
			printStatus("rolled back", "transaction rolled back — all changes undone")
		}
		r.tx = nil
		return
	}

	q, err := query.Parse(input)
	if err != nil {
		printError("parse error: " + err.Error())
		return
	}

	if r.tx != nil {
		r.executeInTx(q)
		return
	}

	r.executeQuery(q)
}

func (r *REPL) executeQuery(q *query.Query) {
	switch q.Type {

	case query.CREATE_TABLE:
		if err := r.db.CreateTable(q.Table, q.Fields); err != nil {
			printError(err.Error())
			return
		}
		cols := "id"
		if len(q.Fields) > 0 {
			cols = strings.Join(q.Fields, ", ")
		}
		printStatus("created", "table '"+q.Table+"' created with columns: "+cols)

	case query.DROP_TABLE:
		if err := r.db.DropTable(q.Table); err != nil {
			printError(err.Error())
			return
		}
		printStatus("dropped", "table '"+q.Table+"' dropped")

	case query.SELECT:
		var records []*engine.Record
		var err error

		if q.WhereKey == "id" {
			rec, e := r.db.Get(q.Table, q.WhereVal)
			if e != nil {
				printError(e.Error())
				return
			}
			records = []*engine.Record{rec}
		} else if q.WhereKey != "" {
			records, err = r.db.Filter(q.Table, q.WhereKey, q.WhereVal)
		} else {
			records, err = r.db.GetAll(q.Table)
		}

		if err != nil {
			printError(err.Error())
			return
		}
		if len(records) == 0 {
			printTable([]string{"result"}, [][]string{{"(no records found)"}})
			return
		}
		cols, rows := recordsToTable(records)
		printTable(cols, rows)

	case query.INSERT:
		id, _ := q.Values["id"].(string)
		if id == "" {
			printError("INSERT requires an 'id' column")
			return
		}
		delete(q.Values, "id")
		if err := r.db.Insert(q.Table, id, q.Values); err != nil {
			printError(err.Error())
			return
		}
		rec, _ := r.db.Get(q.Table, id)
		cols, rows := recordsToTable([]*engine.Record{rec})
		printTable(cols, rows)

	case query.UPDATE:
		if q.WhereKey != "id" {
			printError("UPDATE requires WHERE id = '...'")
			return
		}
		rec, err := r.db.Update(q.Table, q.WhereVal, q.Values)
		if err != nil {
			printError(err.Error())
			return
		}
		cols, rows := recordsToTable([]*engine.Record{rec})
		printTable(cols, rows)

	case query.DELETE:
		if q.WhereKey != "id" {
			printError("DELETE requires WHERE id = '...'")
			return
		}
		if err := r.db.Delete(q.Table, q.WhereVal); err != nil {
			printError(err.Error())
			return
		}
		printStatus("deleted", "record "+q.WhereVal+" removed")
	}
}

func (r *REPL) executeInTx(q *query.Query) {
	switch q.Type {
	case query.INSERT:
		id, _ := q.Values["id"].(string)
		if id == "" {
			printError("INSERT requires an 'id' column")
			return
		}
		delete(q.Values, "id")
		if err := r.tx.TxInsert(q.Table, id, q.Values); err != nil {
			printError(err.Error())
			return
		}
		printStatus("staged", "INSERT staged — type COMMIT to apply or ROLLBACK to cancel")

	case query.UPDATE:
		if q.WhereKey != "id" {
			printError("UPDATE requires WHERE id = '...'")
			return
		}
		if err := r.tx.TxUpdate(q.Table, q.WhereVal, q.Values); err != nil {
			printError(err.Error())
			return
		}
		printStatus("staged", "UPDATE staged — type COMMIT to apply or ROLLBACK to cancel")

	case query.DELETE:
		if q.WhereKey != "id" {
			printError("DELETE requires WHERE id = '...'")
			return
		}
		if err := r.tx.TxDelete(q.Table, q.WhereVal); err != nil {
			printError(err.Error())
			return
		}
		printStatus("staged", "DELETE staged — type COMMIT to apply or ROLLBACK to cancel")

	case query.SELECT:
		r.executeQuery(q)
	}
}

func (r *REPL) handleDotCommand(cmd string) (exit bool) {
	switch strings.ToLower(strings.TrimSpace(cmd)) {

	case ".exit", ".quit":
		return true

	case ".help":
		fmt.Println()
		fmt.Println("Table Commands:")
		fmt.Println("  CREATE TABLE <table> (id, col1, col2)  -- create a new table")
		fmt.Println("  DROP TABLE <table>                     -- delete a table permanently")
		fmt.Println()
		fmt.Println("SQL Commands:")
		fmt.Println("  SELECT * FROM <table>")
		fmt.Println("  SELECT * FROM <table> WHERE id = '<id>'")
		fmt.Println("  SELECT * FROM <table> WHERE <field> = '<value>'")
		fmt.Println("  INSERT INTO <table> (id, field1, field2) VALUES ('v0', 'v1', 'v2')")
		fmt.Println("  UPDATE <table> SET field1='val' WHERE id = '<id>'")
		fmt.Println("  DELETE FROM <table> WHERE id = '<id>'")
		fmt.Println()
		fmt.Println("Transaction Commands:")
		fmt.Println("  BEGIN      -- start a transaction")
		fmt.Println("  COMMIT     -- apply all staged operations")
		fmt.Println("  ROLLBACK   -- undo all staged operations")
		fmt.Println()
		fmt.Println("Shell Commands:")
		fmt.Println("  .tables    -- list all tables")
		fmt.Println("  .help      -- show this help message")
		fmt.Println("  .exit      -- quit GoDB")
		fmt.Println()

	case ".tables":
		tables := r.db.ListTables()
		if len(tables) == 0 {
			fmt.Println("(no tables yet)")
			return false
		}
		rows := make([][]string, len(tables))
		for i, t := range tables {
			rows[i] = []string{t}
		}
		printTable([]string{"table_name"}, rows)

	default:
		printError("unknown command: " + cmd + "  (try .help)")
	}
	return false
}

func printTable(columns []string, rows [][]string) {
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	sep := "+"
	for _, w := range widths {
		sep += strings.Repeat("-", w+2) + "+"
	}

	fmt.Println(sep)

	header := "|"
	for i, col := range columns {
		header += fmt.Sprintf(" %-*s |", widths[i], col)
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, row := range rows {
		line := "|"
		for i := 0; i < len(columns); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line += fmt.Sprintf(" %-*s |", widths[i], cell)
		}
		fmt.Println(line)
	}

	fmt.Println(sep)
	fmt.Printf("%d row(s) returned\n\n", len(rows))
}

func printStatus(status, message string) {
	printTable([]string{"status", "message"}, [][]string{{status, message}})
}

func printError(msg string) {
	fmt.Printf("\nERROR: %s\n\n", msg)
}

func recordsToTable(records []*engine.Record) ([]string, [][]string) {
	keySet := make(map[string]bool)
	for _, rec := range records {
		for k := range rec.Data {
			keySet[k] = true
		}
	}
	dataKeys := make([]string, 0, len(keySet))
	for k := range keySet {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)

	columns := append([]string{"id", "created_at", "updated_at"}, dataKeys...)

	rows := make([][]string, len(records))
	for i, rec := range records {
		row := []string{
			rec.ID,
			rec.CreatedAt.Format("2006-01-02 15:04:05"),
			rec.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		for _, k := range dataKeys {
			row = append(row, fmt.Sprintf("%v", rec.Data[k]))
		}
		rows[i] = row
	}
	return columns, rows
}
