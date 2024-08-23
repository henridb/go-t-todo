package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func first[T, U any](val T, _ U) T {
    return val
}

const create string = `
  CREATE TABLE IF NOT EXISTS todos (
  id INTEGER NOT NULL PRIMARY KEY,
  time DATETIME NOT NULL,
  description TEXT
  );`

const DBfile string = "todos.db"

type Todos struct {
	*sql.DB
}

type Todo struct {
	time time.Time
	description string
}

func createDB() (*Todos, error) {
	db, err := sql.Open("sqlite3", DBfile)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(create); err != nil {
		return nil, err
	}
	return &Todos{db}, nil
}

func (db *Todos) flushDB() error {
	_, err := db.Exec("DELETE FROM todos;")
	return err
}

func (db *Todos) Insert(task Todo) (int, error) {
	res, err := db.Exec("INSERT INTO todos VALUES(NULL,?,?);", task.time, task.description)
	if err != nil {
	 return 0, err
	}
   
	var id int64
	if id, err = res.LastInsertId(); err != nil {
	 return 0, err
	}
	return int(id), nil
}

func (db *Todos) List() ([]Todo, error) {
	rows, err := db.Query(`
    SELECT time, description 
    FROM todos`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []Todo{}
 	for rows.Next() {
		task := Todo{}
		if err = rows.Scan(&task.time, &task.description); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (task Todo) String() string {
	return fmt.Sprintf("%s: %s\n", task.time.Format("01/02 03:04"), task.description)
}

func main() {
	todos, err := createDB()
	todos.flushDB()
	if err != nil {
		fmt.Println("Error!")
		os.Exit(1)
	}
	todos.Insert(Todo{time.Now(), "task 1"})
	todos.Insert(Todo{time.Now(), "task 2"})
	fmt.Println(first(todos.List()))
	subcommands := []string{"add","list","check","remove"}
	for _, cmd := range subcommands {
		flag.NewFlagSet(cmd, flag.ExitOnError)
	}
	if len(os.Args) < 2 || !slices.Contains(subcommands,os.Args[1]) {
		fmt.Println("expected one subcommands of", subcommands)
        os.Exit(1)
	}
}