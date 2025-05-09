package main
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"bytes"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	//"github.com/gorilla/mux"
)

type QueryRequest struct {
	Database string `json:"database"`
	Query    string `json:"query"`
}

var db *sql.DB
var slaveURLs = []string{
	"http://localhost:8081/replicate",
}

func initDB(){
	var err error
	db, err = sql.Open("mysql", "root:07775000a@tcp(127.0.0.1:3306)/")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Connect to MySQL
	initDB()
	defer db.Close()

	http.HandleFunc("/create_db", createDatabase)
	http.HandleFunc("/create_table", createTable)
	http.HandleFunc("/drop_db", dropDatabase)
	http.HandleFunc("/query", handleQuery)

	fmt.Println("[Master] Running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createDatabase(w http.ResponseWriter, r *http.Request) {
	/*if !isMaster{
		http.Error(w, "Unauthorized: Only master can create/drop databases", 403)
		return
	}*/

	var req QueryRequest
	//json.NewDecoder(r.Body).Decode(&req)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("Error Received request:", req)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	} else {
		fmt.Println("Received request:", req)
	}

	_, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + req.Database)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Database created!"))
}

func createTable(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	//json.NewDecoder(r.Body).Decode(&req)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("Error Received request:", req)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	} else {
		fmt.Println("Received request:", req)
	}

	_, err := db.Exec("USE " + req.Database)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = db.Exec(req.Query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Table created!"))
}

func dropDatabase(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	//json.NewDecoder(r.Body).Decode(&req)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("Error Received request:", req)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	} else {
		fmt.Println("Received request:", req)
	}

	_, err := db.Exec("DROP DATABASE " + req.Database)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Database dropped!"))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Block sensitive queries (master-only)
	queryLower := strings.ToLower(req.Query)
	if strings.Contains(queryLower, "create database") || 
		strings.Contains(queryLower, "create table") || 
		strings.Contains(queryLower, "drop database") {
		http.Error(w, "This operation is allowed only via master admin endpoints.", http.StatusForbidden)
		return
	}

	// Check if it's SELECT
	if strings.HasPrefix(strings.ToLower(req.Query), "select") {
		rows, err := db.Query(req.Query)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		cols, _ := rows.Columns()
		result := []map[string]interface{}{}

		for rows.Next() {
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			rowMap := make(map[string]interface{})
			for i, colName := range cols {
				val := columnPointers[i].(*interface{})
				rowMap[colName] = *val
			}

			result = append(result, rowMap)
		}

		json.NewEncoder(w).Encode(result)
		return
	}

	// Other queries (INSERT, UPDATE, DELETE)
	res, err := db.Exec(req.Query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	w.Write([]byte(fmt.Sprintf("Query executed! Rows affected: %d\n", rowsAffected)))

	// Replicate to slaves
	for _, url := range slaveURLs {
		if err := replicateToSlaves(url, req); err != nil {
			http.Error(w, "Replication failed to "+url, http.StatusInternalServerError)
			return
		}	}
}

func replicateToSlaves(url string, req QueryRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		log.Println("Error marshalling query request:", err)
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Println("Replication error to", url, ":", err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("Replicated to %s successfully. Status: %s\n", url, resp.Status)
	return nil
}


