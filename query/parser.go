package query

import (
	"fmt"
	"strings"
)

type QueryType int

const (
	SELECT QueryType = iota
	INSERT
	UPDATE
	DELETE
	CREATE_TABLE
	DROP_TABLE
)

type Query struct {
	Type     QueryType
	Table    string
	Fields   []string               
	Values   map[string]interface{} 
	WhereKey string                 
	WhereVal string                 
	Limit    int
}

func Parse(input string) (*Query, error) {
	input = strings.TrimSpace(input)
	upper := strings.ToUpper(input)

	switch {
	case strings.HasPrefix(upper, "CREATE TABLE"):
		return parseCreateTable(input)
	case strings.HasPrefix(upper, "DROP TABLE"):
		return parseDropTable(input)
	case strings.HasPrefix(upper, "SELECT"):
		return parseSelect(input)
	case strings.HasPrefix(upper, "INSERT"):
		return parseInsert(input)
	case strings.HasPrefix(upper, "UPDATE"):
		return parseUpdate(input)
	case strings.HasPrefix(upper, "DELETE"):
		return parseDelete(input)
	default:
		return nil, fmt.Errorf("unknown query type: %q", input)
	}
}

func parseSelect(s string) (*Query, error) {
	q := &Query{Type: SELECT, Limit: -1}

	rest := strings.TrimSpace(s[6:]) // skip "SELECT"

	fromIdx := strings.Index(strings.ToUpper(rest), " FROM ")
	if fromIdx < 0 {
		return nil, fmt.Errorf("SELECT missing FROM")
	}

	fieldsPart := strings.TrimSpace(rest[:fromIdx])
	rest = strings.TrimSpace(rest[fromIdx+6:])

	if fieldsPart == "*" {
		q.Fields = []string{"*"}
	} else {
		for _, f := range strings.Split(fieldsPart, ",") {
			q.Fields = append(q.Fields, strings.TrimSpace(f))
		}
	}

	parts := strings.SplitN(strings.ToUpper(rest), " WHERE ", 2)
	q.Table = strings.TrimSpace(strings.ToLower(parts[0]))

	if len(parts) == 2 {
		whereStart := strings.Index(strings.ToUpper(rest), " WHERE ") + 7
		whereKey, whereVal, err := parseWhere(rest[whereStart:])
		if err != nil {
			return nil, err
		}
		q.WhereKey = whereKey
		q.WhereVal = whereVal
	}
	return q, nil
}

func parseInsert(s string) (*Query, error) {
	q := &Query{Type: INSERT, Values: make(map[string]interface{})}

	// skip "INSERT INTO "
	rest := strings.TrimSpace(s[12:])

	spaceIdx := strings.Index(rest, " ")
	if spaceIdx < 0 {
		return nil, fmt.Errorf("INSERT syntax error: missing table name")
	}
	q.Table = strings.ToLower(rest[:spaceIdx])
	rest = strings.TrimSpace(rest[spaceIdx:])

	colStart := strings.Index(rest, "(")
	colEnd := strings.Index(rest, ")")
	if colStart < 0 || colEnd < 0 {
		return nil, fmt.Errorf("INSERT missing column list")
	}
	cols := strings.Split(rest[colStart+1:colEnd], ",")
	for i := range cols {
		cols[i] = strings.TrimSpace(cols[i])
	}

	rest = strings.TrimSpace(rest[colEnd+1:])
	valIdx := strings.Index(strings.ToUpper(rest), "VALUES")
	if valIdx < 0 {
		return nil, fmt.Errorf("INSERT missing VALUES keyword")
	}
	rest = strings.TrimSpace(rest[valIdx+6:])

	valStart := strings.Index(rest, "(")
	valEnd := strings.LastIndex(rest, ")")
	if valStart < 0 || valEnd < 0 {
		return nil, fmt.Errorf("INSERT missing value list")
	}
	vals := strings.Split(rest[valStart+1:valEnd], ",")
	if len(cols) != len(vals) {
		return nil, fmt.Errorf("column/value count mismatch: %d columns, %d values", len(cols), len(vals))
	}
	for i, col := range cols {
		q.Values[col] = strings.Trim(strings.TrimSpace(vals[i]), `'"`)
	}
	return q, nil
}

func parseUpdate(s string) (*Query, error) {
	q := &Query{Type: UPDATE, Values: make(map[string]interface{})}

	rest := strings.TrimSpace(s[6:]) // skip "UPDATE"

	spaceIdx := strings.Index(rest, " ")
	if spaceIdx < 0 {
		return nil, fmt.Errorf("UPDATE syntax error: missing table name")
	}
	q.Table = strings.ToLower(rest[:spaceIdx])
	rest = strings.TrimSpace(rest[spaceIdx:])

	setIdx := strings.Index(strings.ToUpper(rest), "SET ")
	if setIdx < 0 {
		return nil, fmt.Errorf("UPDATE missing SET")
	}
	rest = strings.TrimSpace(rest[setIdx+4:])

	whereIdx := strings.Index(strings.ToUpper(rest), " WHERE ")
	var setPart string
	if whereIdx >= 0 {
		setPart = rest[:whereIdx]
		whereKey, whereVal, err := parseWhere(rest[whereIdx+7:])
		if err != nil {
			return nil, err
		}
		q.WhereKey = whereKey
		q.WhereVal = whereVal
	} else {
		setPart = rest
	}

	for _, assignment := range strings.Split(setPart, ",") {
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid SET clause: %q", assignment)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `'"`)
		q.Values[key] = val
	}
	return q, nil
}

func parseDelete(s string) (*Query, error) {
	q := &Query{Type: DELETE}

	// skip "DELETE FROM "
	rest := strings.TrimSpace(s[11:])

	whereIdx := strings.Index(strings.ToUpper(rest), " WHERE ")
	if whereIdx >= 0 {
		q.Table = strings.ToLower(strings.TrimSpace(rest[:whereIdx]))
		whereKey, whereVal, err := parseWhere(rest[whereIdx+7:])
		if err != nil {
			return nil, err
		}
		q.WhereKey = whereKey
		q.WhereVal = whereVal
	} else {
		q.Table = strings.ToLower(strings.TrimSpace(rest))
	}
	return q, nil
}

func parseWhere(s string) (string, string, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid WHERE clause: %q", s)
	}
	key := strings.TrimSpace(parts[0])
	val := strings.Trim(strings.TrimSpace(parts[1]), `'"`)
	return key, val, nil
}

func parseCreateTable(s string) (*Query, error) {
	q := &Query{Type: CREATE_TABLE}

	rest := strings.TrimSpace(s[12:])

	parenIdx := strings.Index(rest, "(")
	if parenIdx >= 0 {
		q.Table = strings.ToLower(strings.TrimSpace(rest[:parenIdx]))
		closeIdx := strings.Index(rest, ")")
		if closeIdx < 0 {
			return nil, fmt.Errorf("CREATE TABLE missing closing ')'")
		}
		cols := strings.Split(rest[parenIdx+1:closeIdx], ",")
		for _, c := range cols {
			c = strings.TrimSpace(c)
			if c != "" {
				q.Fields = append(q.Fields, c)
			}
		}
	} else {
		q.Table = strings.ToLower(strings.TrimSpace(rest))
	}

	if q.Table == "" {
		return nil, fmt.Errorf("CREATE TABLE requires a table name")
	}
	return q, nil
}

func parseDropTable(s string) (*Query, error) {
	q := &Query{Type: DROP_TABLE}
	// skip "DROP TABLE "
	rest := strings.TrimSpace(s[10:])
	q.Table = strings.ToLower(strings.TrimSpace(rest))
	if q.Table == "" {
		return nil, fmt.Errorf("DROP TABLE requires a table name")
	}
	return q, nil
}
