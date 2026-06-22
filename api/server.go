package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"github.com/arijiiiitttt/dinoDB/engine"
	"github.com/arijiiiitttt/dinoDB/query"
)

type Server struct {
	db   *engine.DB
	mux  *http.ServeMux
	addr string
}

// NewServer creates a new HTTP server 
func NewServer(db *engine.DB, addr string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), addr: addr}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/records/", s.handleRecords)
	s.mux.HandleFunc("/query", s.handleQuery)
	s.mux.HandleFunc("/tables", s.handleTables)
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	fmt.Printf("dinoDB HTTP server listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func renderTable(w http.ResponseWriter, columns []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

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

	fmt.Fprintln(w, sep)
	header := "|"
	for i, col := range columns {
		header += fmt.Sprintf(" %-*s |", widths[i], col)
	}
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, sep)

	for _, row := range rows {
		line := "|"
		for i := 0; i < len(columns); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line += fmt.Sprintf(" %-*s |", widths[i], cell)
		}
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "%d row(s) returned\n", len(rows))
}

func recordsToTable(records []*engine.Record) ([]string, [][]string) {
	keySet := make(map[string]bool)
	for _, r := range records {
		for k := range r.Data {
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
	for i, r := range records {
		row := []string{
			r.ID,
			r.CreatedAt.Format("2006-01-02 15:04:05"),
			r.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		for _, k := range dataKeys {
			row = append(row, fmt.Sprintf("%v", r.Data[k]))
		}
		rows[i] = row
	}
	return columns, rows
}

func statusTable(w http.ResponseWriter, status, message string) {
	renderTable(w, []string{"status", "message"}, [][]string{{status, message}})
}

func errorTable(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	renderTable(w, []string{"error"}, [][]string{{msg}})
}

func (s *Server) handleRecords(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/records/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		errorTable(w, "table name required in URL", http.StatusBadRequest)
		return
	}
	table := parts[0]
	id := ""
	if len(parts) >= 2 {
		id = parts[1]
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r, table, id)
	case http.MethodPost:
		s.handleInsert(w, r, table)
	case http.MethodPut:
		s.handleUpdate(w, r, table, id)
	case http.MethodDelete:
		s.handleDelete(w, r, table, id)
	default:
		errorTable(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, table, id string) {
	var records []*engine.Record
	if id == "" {
		var err error
		records, err = s.db.GetAll(table)
		if err != nil {
			errorTable(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		rec, err := s.db.Get(table, id)
		if err != nil {
			errorTable(w, err.Error(), http.StatusNotFound)
			return
		}
		records = []*engine.Record{rec}
	}
	if len(records) == 0 {
		renderTable(w, []string{"result"}, [][]string{{"(no records found)"}})
		return
	}
	cols, rows := recordsToTable(records)
	renderTable(w, cols, rows)
}

func (s *Server) handleInsert(w http.ResponseWriter, r *http.Request, table string) {
	var body struct {
		ID   string                 `json:"id"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorTable(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.ID == "" {
		errorTable(w, "id is required", http.StatusBadRequest)
		return
	}
	if err := s.db.Insert(table, body.ID, body.Data); err != nil {
		errorTable(w, err.Error(), http.StatusConflict)
		return
	}
	rec, _ := s.db.Get(table, body.ID)
	w.WriteHeader(http.StatusCreated)
	cols, rows := recordsToTable([]*engine.Record{rec})
	renderTable(w, cols, rows)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request, table, id string) {
	if id == "" {
		errorTable(w, "id is required in URL", http.StatusBadRequest)
		return
	}
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		errorTable(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	rec, err := s.db.Update(table, id, updates)
	if err != nil {
		errorTable(w, err.Error(), http.StatusNotFound)
		return
	}
	cols, rows := recordsToTable([]*engine.Record{rec})
	renderTable(w, cols, rows)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, table, id string) {
	if id == "" {
		errorTable(w, "id is required in URL", http.StatusBadRequest)
		return
	}
	if err := s.db.Delete(table, id); err != nil {
		errorTable(w, err.Error(), http.StatusNotFound)
		return
	}
	statusTable(w, "deleted", "record "+id+" removed successfully")
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorTable(w, "use POST with JSON body {\"sql\":\"...\"}", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SQL string `json:"sql"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorTable(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	q, err := query.Parse(body.SQL)
	if err != nil {
		errorTable(w, "parse error: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch q.Type {
	case query.SELECT:
		var records []*engine.Record
		if q.WhereKey == "id" {
			rec, err := s.db.Get(q.Table, q.WhereVal)
			if err != nil {
				errorTable(w, err.Error(), http.StatusNotFound)
				return
			}
			records = []*engine.Record{rec}
		} else if q.WhereKey != "" {
			records, err = s.db.Filter(q.Table, q.WhereKey, q.WhereVal)
		} else {
			records, err = s.db.GetAll(q.Table)
		}
		if err != nil {
			errorTable(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(records) == 0 {
			renderTable(w, []string{"result"}, [][]string{{"(no records found)"}})
			return
		}
		cols, rows := recordsToTable(records)
		renderTable(w, cols, rows)

	case query.INSERT:
		id, _ := q.Values["id"].(string)
		delete(q.Values, "id")
		if err := s.db.Insert(q.Table, id, q.Values); err != nil {
			errorTable(w, err.Error(), http.StatusConflict)
			return
		}
		statusTable(w, "inserted", "record "+id+" created successfully")

	case query.UPDATE:
		if q.WhereKey != "id" {
			errorTable(w, "UPDATE requires WHERE id = '...'", http.StatusBadRequest)
			return
		}
		rec, err := s.db.Update(q.Table, q.WhereVal, q.Values)
		if err != nil {
			errorTable(w, err.Error(), http.StatusNotFound)
			return
		}
		cols, rows := recordsToTable([]*engine.Record{rec})
		renderTable(w, cols, rows)

	case query.DELETE:
		if q.WhereKey != "id" {
			errorTable(w, "DELETE requires WHERE id = '...'", http.StatusBadRequest)
			return
		}
		if err := s.db.Delete(q.Table, q.WhereVal); err != nil {
			errorTable(w, err.Error(), http.StatusNotFound)
			return
		}
		statusTable(w, "deleted", "record "+q.WhereVal+" removed successfully")
	}
}

func (s *Server) handleTables(w http.ResponseWriter, r *http.Request) {
	tables := s.db.ListTables()
	rows := make([][]string, len(tables))
	for i, t := range tables {
		rows[i] = []string{t, fmt.Sprintf("table #%d", i+1)}
	}
	renderTable(w, []string{"table_name", "info"}, rows)
}
