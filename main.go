package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var helpDoc = "Get help"

var subcommandsDoc = map[string]string{
	"add": "Add a new task",
	"list": "List all tasks",
	"check": "Check a task",
	"remove": "Remove a task",
	"help": helpDoc,
}

var subcommands = func() []string {
	res := make([]string, 0, len(subcommandsDoc))
	for cmd := range subcommandsDoc {
		res = append(res, cmd)
	}
	return res
}()


func first[T, U any](val T, _ U) T {
    return val
}

const create string = `
  CREATE TABLE IF NOT EXISTS todos (
  id INTEGER NOT NULL PRIMARY KEY,
  time DATETIME NOT NULL,
  description TEXT,
  checked BOOLEAN DEFAULT false
  );`

const DBfile string = "todos.db"

type Todos struct {
	*sql.DB
}

type Todo struct {
	time time.Time
	description string
	checked bool
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
	res, err := db.Exec("INSERT INTO todos VALUES(NULL,?,?,?);", task.time, task.description, task.checked)
	if err != nil {
	 return 0, err
	}
   
	var id int64
	if id, err = res.LastInsertId(); err != nil {
	 return 0, err
	}
	return int(id), nil
}

func (db *Todos) List(filterUnchecked bool) ([]Todo, error) {
	filter := ""
	if filterUnchecked {
		filter = " WHERE checked = 0"
	}
	rows, err := db.Query(`
    SELECT time, description, checked
    FROM todos` + filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []Todo{}
 	for rows.Next() {
		task := Todo{}
		if err = rows.Scan(&task.time, &task.description, &task.checked); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (task Todo) String() string {
	checked := "[ ] "
	if task.checked {
		checked = "[x] "
	}
	return fmt.Sprintf("%s%s: %s", checked, task.time.Format("01/02 03:04"), task.description)
}

func main() {
	todos, err := createDB()
	if err != nil {
		fmt.Println("Error!")
		os.Exit(1)
	}

	// todos.flushDB()
	// todos.Insert(Todo{time.Now(), "task 1", false})
	// todos.Insert(Todo{time.Now(), "task 2", true})
	// fmt.Println(first(todos.List()))
	listUncheckedDoc := "Only display the tasks that are not checked"
	h, hLong := flag.Bool("h", false, helpDoc), flag.Bool("help", false, helpDoc)
	flagSets := map[string]*flag.FlagSet{}
	for _, cmd := range subcommands {
		flagSets[cmd] = flag.NewFlagSet(cmd, flag.ExitOnError)
	}
	u := flagSets["list"].Bool("u", false, listUncheckedDoc)
	uLong := flagSets["list"].Bool("unchecked-only", false, listUncheckedDoc)
	flag.Parse()

	if *h || *hLong {
		moduleDoc()
	}
	if len(os.Args) < 2 || !slices.Contains(subcommands,os.Args[1]) {
		fmt.Print("Expected one subcommands of `", strings.Join(subcommands, "`, `"), "`\n")
        os.Exit(1)
	}
	switch os.Args[1] {
	case "add":
		todos.Insert(Todo{time.Now(), os.Args[2], false})
	case "list":
		listTasks(*u || *uLong, *todos)
	case "check":
	case "remove":
	case "help":
		moduleDoc()
	default:
		fmt.Println("This branch should be unreachable...")
		os.Exit(1)
	}
}

func listTasks(filterUnchecked bool, todos Todos) {
	tasks, err := todos.List(filterUnchecked)
	if err != nil {
		fmt.Println("Err: ", err)
		os.Exit(1)
	}
	for _, task := range tasks {
		fmt.Println(task)
	}
	// fmt.Println(first(todos.List(filterUnchecked)))
}

func moduleDoc() {
	fmt.Println("List of available subcommands:")

	for cmd, doc := range subcommandsDoc {
		print("  ", cmd, ": ", doc, "\n")
	}
	os.Exit(0)
}