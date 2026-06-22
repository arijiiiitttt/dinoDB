
<p align="center">
  <a>
    <img alt="dinoDB logo" src="public/dinoDB.png" width="134">
  </a>
</p>

<p align="center">
  <a href="https://discord.com/users/thearijiiiitttt_"><img alt="Discord" src="https://img.shields.io/badge/discord-community-5865F2?style=flat-square&logo=discord&logoColor=white" /></a>
</p>

<p align="center">
  <b>dinoDB</b> Simple embedded database in Go 🐘
</p>

> A lightweight, fun, and educational embedded database with HTTP API, beautiful REPL, WAL durability, and B-Tree indexing.

## Table of Contents
1. [What is dinoDB?](#-what-is-dinodb)
2. [Features](#-features)
3. [Quick Start](#-quick-start)
4. [Interactive REPL](#-interactive-repl)
5. [HTTP REST API](#-http-rest-api)
6. [Project Structure](#-project-structure)
7. [Architecture Highlights](#-architecture-highlights)
8. [Available Commands](#-available-commands)
9. [Environment & Docker](#-environment--docker)
10. [Current Limitations](#-current-limitations)
11. [Roadmap](#-roadmap)
12. [Troubleshooting](#-troubleshooting)

<br/>

##  What is dinoDB?

**dinoDB** is a simple embedded database written in **Golang**. It is designed for learning, prototyping, and small applications. You can use it via a **beautiful terminal REPL** or a **REST HTTP API**.

It supports basic SQL-like operations, persistence with WAL, and B-Tree indexing.

<br/>

##  Features

- SQL-like query support (CREATE, SELECT, INSERT, UPDATE, DELETE, DROP)
- Beautiful colored REPL with Unicode tables
- Full HTTP REST API
- Write-Ahead Logging (WAL) for durability & crash recovery
- B-Tree indexing for fast lookups
- Basic transaction support (`BEGIN`, `COMMIT`, `ROLLBACK`)
- Docker support
- ASCII Dino art on startup

<br/>

## Quick Start

### 1. Clone & Run with Go

```bash
git clone https://github.com/arijiiiitttt/dinoDB.git
cd dinoDB
go run .
```

### 2. Using Docker

```bash
docker-compose up --build
```

The server will be available at **http://localhost:8080**

<br/>

##  Interactive REPL

After starting you will see the dino and enter the REPL:

```bash
dinoDB> CREATE TABLE users (name, age, email)
dinoDB> INSERT INTO users (id, name, age) VALUES ('u1', 'Alice', '30')
dinoDB> SELECT * FROM users
dinoDB> .tables
```


### Available Commands

| Type               | Example                                              | Description                     |
|--------------------|------------------------------------------------------|---------------------------------|
| Table              | `CREATE TABLE users (...)`                           | Create table                    |
|                    | `DROP TABLE users`                                   | Drop table                      |
| CRUD               | `SELECT * FROM users`                                | Select records                  |
|                    | `INSERT INTO users ... VALUES ...`                   | Insert record                   |
|                    | `UPDATE users SET age='31' WHERE id = 'u1'`          | Update record                   |
|                    | `DELETE FROM users WHERE id = 'u1'`                  | Delete record                   |
| Transactions       | `BEGIN` / `COMMIT` / `ROLLBACK`                      | Manage transactions             |
| Shell              | `.tables` / `.help` / `.exit`                        | Utility commands                |

<br/>

##  HTTP REST API

**Base URL**: `http://localhost:8080`

| Method | Endpoint                     | Description                    |
|--------|------------------------------|--------------------------------|
| GET    | `/records/{table}`           | Get all records                |
| GET    | `/records/{table}/{id}`      | Get one record                 |
| POST   | `/records/{table}`           | Create record                  |
| PUT    | `/records/{table}/{id}`      | Update record                  |
| DELETE | `/records/{table}/{id}`      | Delete record                  |
| GET    | `/tables`                    | List tables                    |
| POST   | `/query`                     | Run SQL query                  |

<br/>

**Example:**
```bash
curl -X POST http://localhost:8080/records/users \
  -H "Content-Type: application/json" \
  -d '{"id":"u1","data":{"name":"Bob","age":25}}'
```

<br/>

## Project Structure

```
dinoDB/
├── main.go
├── api/server.go
├── engine/          # Core DB logic + transactions
├── query/           # SQL parser
├── index/btree.go
├── storage/         # WAL + Disk manager
├── repl/repl.go
├── public/dinoDB.png
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

<br/>

##  Architecture Highlights

- WAL for durability
- B-Tree for indexing
- RWMutex for concurrency safety
- Automatic recovery on startup
- Simple document-style records

<br/>

##  Environment & Docker

No special environment variables needed.

Docker volume `/data` is used for persistence.

<br/>

##  Current Limitations

- Basic query parser (no complex WHERE, no JOINs)
- Only exact match filtering supported
- Transactions are basic
- Disk storage is still being improved

<br/>


##  Troubleshooting

| Problem                        | Solution                                      |
|--------------------------------|-----------------------------------------------|
| REPL not showing properly      | Use modern terminal (supports Unicode)        |
| Data not saving                | Check permissions in `./dinoDB` folder        |
| Docker issues                  | `docker-compose up --build --force-recreate`  |
| API not responding             | Make sure server is running on port 8080      |

<br/>

<p align="center">
<b>
Contributions and feedback are welcome!
</b>
</p>
