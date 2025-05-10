
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
   ```
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

| Endpoint     | Method | Description                  |
| ------------ | ------ | ---------------------------- |
| `/replicate` | POST   | Receives and applies a query |
| `/query`     | POST   |                              |

---

## Sending Requests

Use Postman or `curl` to send HTTP POST requests.

**Create DB**

```json
POST /create_db
{
  "database": "testdb"
}
```

**Create Table**

```json
curl -X POST http://localhost:8080/query \
     -H "Content-Type: application/json" \
     -d '{"database": "testdb", "query": "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"}'
```

**Insert Data**

```json
POST /query
{
  "database": "testdb",
  "query": "INSERT INTO users (id, name) VALUES (1, 'Aya')"
}
```

**Drop DB**

```json
POST /drop_db
{
  "database": "testdb"
}
```

---

## Notes

* Slave nodes **must not send queries** directly to the DB. They should only listen to and apply replicated queries.
* For local testing, you can run Master and multiple Slaves on different ports on the same machine.

---

## Author

Aya Abdellatif
