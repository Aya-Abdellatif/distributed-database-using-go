package main

import (
	//"bytes"
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
	//IsMaster bool   `json:"is_master"`
}

var db *sql.DB
var slaveLogger *log.Logger
var masterAddress = "http://localhost:8000"

func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:slave_db_pass@tcp(127.0.0.1:3306)/")
	if err != nil {
		log.Fatal(err)
	}
}

/*func initLogger() {
	logFile, err := os.OpenFile("slave.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	slaveLogger = log.New(logFile, "SLAVE LOG: ", log.Ldate|log.Ltime|log.Lshortfile)
}*/

func isDatabaseOperation(query string) bool {
	queryLower := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(queryLower, "create database") ||
		strings.HasPrefix(queryLower, "create table") ||
		strings.HasPrefix(queryLower, "drop database") ||
		strings.HasPrefix(queryLower, "drop table")
}

/*func isWriteOperation(query string) bool {
	queryLower := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(queryLower, "insert") ||
		strings.HasPrefix(queryLower, "update") ||
		strings.HasPrefix(queryLower, "delete")
}
*/
func main() {
	//initLogger()
	initDB()
	defer db.Close()

	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/replicate", handleReplication)

	slaveLogger.Println("[Slave] Running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	fmt.Println("Received request:", req)


	if isDatabaseOperation(req.Query){
		http.Error(w, "Slave is not supported or authorized to run such database operation.", http.StatusBadRequest)
		return
	}

	//req.IsMaster = false
	queryLower := strings.TrimSpace(strings.ToLower(req.Query))
	if strings.HasPrefix(queryLower, "select") {
		handleSelectQuery(w, req.Query)
	} else{
		body := strings.NewReader(fmt.Sprintf(`{"query": %q}`, req.Query))
		resp, err := http.Post(masterAddress+"/query", "application/json", body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
		slaveLogger.Printf("Error executing replicated query: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	slaveLogger.Printf("Replicated query executed. Rows affected: %d\n", rowsAffected)
	w.Write([]byte(fmt.Sprintf("Replicated query executed. Rows affected: %d\n", rowsAffected)))
}
