// Master Node (master.go)updated
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	//"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type QueryRequest struct {
	Database string `json:"database"`
	Query    string `json:"query"`
}

var db *sql.DB
var slaveURLs = []string{
	"http://192.168.50.36:8081/replicate",
	//"http://localhost:8082/replicate",
}
//var masterLogger *log.Logger

func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:rootroot@tcp(127.0.0.1:3306)/")
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
}

/*func initLogger() {
	logFile, err := os.OpenFile("master.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("Error opening log file:", err)
	}
	masterLogger = log.New(logFile, "MASTER: ", log.Ldate|log.Ltime|log.Lshortfile)
}*/

func isDatabaseOperation(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(q, "create database") || strings.HasPrefix(q, "drop database") ||
		strings.HasPrefix(q, "create table") || strings.HasPrefix(q, "drop table")
}

func isWriteOperation(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(q, "insert") || strings.HasPrefix(q, "update") || strings.HasPrefix(q, "delete")
}

func main() {
	//initLogger()
	initDB()
	defer db.Close()

	http.HandleFunc("/query", handleQuery)
	log.Println("Master running at :8080")
	http.ListenAndServe(":8080", nil)
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	fmt.Println("Received request:", req)

	if req.Database != "" {
		if _, err := db.Exec("USE " + req.Database); err != nil {
			http.Error(w, "Database not found", http.StatusBadRequest)
			return
		}
	}

	query := strings.TrimSpace(strings.ToLower(req.Query))
	if strings.HasPrefix(query, "select") {
		handleSelectQuery(w, req.Query)
	} else if isWriteOperation(req.Query) || isDatabaseOperation(req.Query){// && req.IsMaster) {
		handleWriteAndReplicate(w, req)
	} else {
		http.Error(w, "Unsupported or unauthorized command", http.StatusBadRequest)
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

func handleWriteAndReplicate(w http.ResponseWriter, req QueryRequest) {
	res, err := db.Exec(req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	fmt.Fprintf(w, "Query executed. Rows affected: %d", rowsAffected)
	replicateToSlaves(req)
}

func replicateToSlaves(req QueryRequest) {
	data, err := json.Marshal(req)
	if err != nil {
		log.Println("Error marshalling query request:", err)
		return
	}

	for _, url := range slaveURLs {
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Println("Replication error to", url, ":", err)
			return 
		}
		defer resp.Body.Close()

		log.Printf("Replicated to %s successfully. Status: %s\n", url, resp.Status)
	}
}
