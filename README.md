
# Distributed Database System

This project implements a basic distributed database system in Go with a **Master-Slave architecture**. The Master node manages database creation, table creation, general query execution, and replicates queries to Slave nodes.

---

## Architecture Overview

* **One Master Node**

  * Full DB write access.
  * Executes queries and replicates them to all slaves.

* **Multiple Slave Nodes**

  * Handle read-only queries.
  * Write access only through replication from master.
  * Cannot initiate create/drop operations themselves.
  * Listen for replicated queries and apply them locally.
  
 The communication between Master and Slaves is done using HTTP.

---
## System Architecture Diagram

                 +-----------------------------+
                 |         Master Node         |
                 |-----------------------------|
                 | - Write Access (POST /query)|
                 | - Broadcast to Slaves       |
                 +-----------------------------+
                           |
                    +------+------+
                    |             |
                    ▼             ▼
            +-----------------+   +-----------------+
            |    Slave Node   |   |    Slave Node   |
            |-----------------|   |-----------------|
            | - Read-only DB  |   | - Read-only DB  |
            | - Listen (POST) |   | - Listen (POST) |
            | - Read Access   |   | - Read Access   |
            +-----------------+   +-----------------+

---

## Folder Structure

```
distributed-database-using-go/
├── master/
│   └── master.go
├── slave/
│   └── slave.go
├── README.md
└── report.pdf
```

---

## Prerequisites

* MySQL server installed and running.
* MySQL user with access rights (e.g., `root:password`).
* Go installed on your system.

---

## Running the System
1. **Install dependencies:**
   ```bash
   go mod tidy
   ```
   
2. **Start the Master node:**

```bash
cd master
go run master.go
```

Runs at: `http://localhost:8080`

3. **Start one or more Slave nodes (on different ports):**

```bash
cd slave
# Terminal 1
PORT=8081 go run node.go
# Terminal 2
PORT=8082 go run node.go
```

You can adjust the listening port inside `node/node.go`.

---

## API Endpoints

### Master

| Endpoint        | Method | Description                      |
| --------------- | ------ | -------------------------------- |
| `/query`        | POST   | Execute a query and replicate it |

### Slave

| Endpoint     | Method | Description                                   |
| ------------ | ------ | ----------------------------------------------|
| `/replicate` | POST   | Receives and applies a query                  |
| `/query`     | POST   | Execute a read-only query (optional endpoint) |


---

## Sending Requests

Use Postman or `curl` to send HTTP POST requests.

**Create DB**

```bash
curl -X POST http://localhost:8080/query \
     -H "Content-Type: application/json" \
     -d '{\"database\": \"\", \"query\": \"CREATE DATABASE company\"}'
```

**Create Table**

```bash
curl -X POST http://localhost:8080/query \
     -H "Content-Type: application/json" \
     -d '{\"database\": \"company\", \"query\": \"CREATE TABLE employee (id INT PRIMARY KEY, name VARCHAR(50))\"}'
```

**Insert Data**

```bash
curl -X POST http://localhost:8080/query \
     -H "Content-Type: application/json" \
     -d '{\"database\": \"company\", \"query\": \"INSERT INTO employee (id, name) VALUES (1, \"Aya\")\"}'

```

---

## Author

Aya Abdellatif
