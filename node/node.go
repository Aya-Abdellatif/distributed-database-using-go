package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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

	http.HandleFunc("/replicate", replicateQuery)
    http.HandleFunc("/forward_to_master", forwardToMaster)

	fmt.Println("Slave running on :8081")
	http.ListenAndServe(":8081", nil)
}

func replicateQuery(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req QueryRequest
	json.Unmarshal(body, &req)

	// Block sensitive queries
	queryLower := strings.ToLower(req.Query)
	if strings.HasPrefix(queryLower, "create database") ||
		strings.HasPrefix(queryLower, "drop database") ||
		strings.HasPrefix(queryLower, "create table") {
		http.Error(w, "This operation is master-only. Replication rejected.", http.StatusForbidden)
		return
	}

	db.Exec("USE " + req.Database)
	_, err := db.Exec(req.Query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	log.Println("Query replicated:", req.Query)
	w.Write([]byte("Query applied on slave."))
}

func forwardToMaster(w http.ResponseWriter, r *http.Request) {
	masterURL := "http://localhost:8080/query"
	resp, err := http.Post(masterURL, "application/json", r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}
