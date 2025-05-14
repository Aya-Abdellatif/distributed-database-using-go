package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type QueryRequest struct {
	Database string `json:"database"`
	Query    string `json:"query"`
}

var db *sql.DB
var masterAddress = "http://192.168.50.161:8080" // تأكد إن دا IP فعلي للماستر في الشبكة

func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:rootroot@tcp(127.0.0.1:3306)/")
	if err != nil {
		log.Fatal("Failed to connect to slave DB:", err)
	}
}

func isDatabaseOperation(query string) bool {
	queryLower := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(queryLower, "create database") ||
		strings.HasPrefix(queryLower, "create table") ||
		strings.HasPrefix(queryLower, "drop database") ||
		strings.HasPrefix(queryLower, "drop table")
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/replicate", handleReplication)

	log.Println("[Slave] Running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	fmt.Println("Received request:", req)

	if isDatabaseOperation(req.Query) {
		http.Error(w, "Slave not authorized to execute DDL queries", http.StatusForbidden)
		return
	}

	// Forward write queries to master
	queryLower := strings.TrimSpace(strings.ToLower(req.Query))
	if strings.HasPrefix(queryLower, "select") {
		handleSelectQuery(w, req.Query)
	} else {
		// Forward to master
		jsonData, _ := json.Marshal(req)
		resp, err := http.Post(masterAddress+"/query", "application/json", strings.NewReader(string(jsonData)))
		if err != nil {
			http.Error(w, "Failed to forward to master: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Master error", resp.StatusCode)
			return
		}

		w.Write([]byte("Query forwarded to master."))
	}
}

func handleSelectQuery(w http.ResponseWriter, query string) {
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	results := []map[string]interface{}{}

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		if err := rows.Scan(columnPointers...); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = *(columnPointers[i].(*interface{}))
		}
		results = append(results, row)
	}
	json.NewEncoder(w).Encode(results)
}

func handleReplication(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid replication request", http.StatusBadRequest)
		return
	}

	if req.Database != "" {
		if _, err := db.Exec("USE " + req.Database); err != nil {
			http.Error(w, "Database not found", http.StatusBadRequest)
			return
		}
	}

	res, err := db.Exec(req.Query)
	if err != nil {
		log.Printf("Error executing replicated query: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	log.Printf("Replicated query executed. Rows affected: %d\n", rowsAffected)
	w.Write([]byte(fmt.Sprintf("Replicated query executed. Rows affected: %d\n", rowsAffected)))
}
