
# Distributed Database System

This project implements a basic distributed database system in Go with a **Master-Slave architecture**. The Master node manages database creation, table creation, general query execution, and replicates queries to Slave nodes.

---

## Architecture Overview

* **Master Node**

  * Full DB write access.
  * Can create and drop databases and tables.
  * Executes queries and replicates them to all slaves.

* **Slave Nodes**

  * Read/Write access only through replication from master.
  * Cannot initiate create/drop operations themselves.
  * Listen for replicated queries and apply them locally.

---

## Project Structure

```
distributed-db/
├── master/
│   └── master.go
├── node/
│   └── node.go
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
| `/create_db`    | POST   | Create a new database            |
| `/create_table` | POST   | Create a table in a database     |
| `/drop_db`      | POST   | Drop a database                  |
| `/query`        | POST   | Execute a query and replicate it |

### Slave

| Endpoint     | Method | Description                  |
| ------------ | ------ | ---------------------------- |
| `/replicate` | POST   | Receives and applies a query |

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
POST /create_table
{
  "database": "testdb",
  "query": "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))"
}
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
