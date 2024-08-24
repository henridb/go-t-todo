package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var helpDoc = "Get help"

var subcommandsDoc = map[string]string{
	"add": "Add a new task",
	"list": "List all tasks",
	"toggle": "Toggle the check state of a task",
	"remove": "Remove a task",
	"help": helpDoc,
}

var subcommands = func() []string {
	res := make([]string, 0, len(subcommandsDoc))
	for cmd := range subcommandsDoc {
		res = append(res, cmd)
	}
	sort.Strings(res)
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
	id int
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

func (db *Todos) flush() error {
	_, err := db.Exec("DELETE FROM todos;")
	return err
}

func (db *Todos) insert(description string) error {
	_, err := db.Exec("INSERT INTO todos VALUES(NULL,?,?,?);", time.Now(), description, false)
	return err
}

func (db *Todos) execInsert(description string) {
	err := db.insert(description)
	if err != nil {
		fmt.Println("Err:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func (db *Todos) list(filterUnchecked bool) ([]Todo, error) {
	filter := ""
	if filterUnchecked {
		filter = " WHERE checked = 0"
	}
	rows, err := db.Query(`
    SELECT time, description, checked, id
    FROM todos` + filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []Todo{}
 	for rows.Next() {
		task := Todo{}
		if err = rows.Scan(&task.time, &task.description, &task.checked, &task.id); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (db *Todos) execList(filterUnchecked bool) {
	tasks, err := db.list(filterUnchecked)
	if err != nil {
		fmt.Println("Err: ", err)
		os.Exit(1)
	}
	for _, task := range tasks {
		fmt.Println(task)
	}
	os.Exit(0)
}

func (db *Todos) toggleTask(id int) error {
	_, err := db.Exec("UPDATE todos SET checked = NOT checked WHERE id = ?", id)
	return err
}

func (db *Todos) execToggleTask() {
	tasks, err := db.list(false)
	if err != nil {
		fmt.Println("Err: ", err)
		os.Exit(1)
	}
	maxIndexLen := int(math.Log10(float64(len(tasks))))

	fmt.Println("Select task to check:")
	for i, task := range tasks {
		fmt.Printf("%*d) %s\n", maxIndexLen+1, i, task)
	}
	var lineNb int
	fmt.Scan(&lineNb)
	
	id := tasks[lineNb].id 
	err = db.toggleTask(id)
	if err != nil {
		fmt.Println("Err: ", err)
		os.Exit(1)
	}
	fmt.Print("Task '", tasks[lineNb].description, "' toggled\n")
}



func (task Todo) String() string {
	checked := "[ ] "
	if task.checked {
		checked = "[x] "
	}
	return fmt.Sprintf("%s%s: %s", checked, task.time.Format("01/02 03:04"), task.description)
}

func main() {
	var mainHelp bool
	shorthandedFlag(flag.BoolVar, &mainHelp, "help", false, helpDoc)
	flagSets := map[string]*flag.FlagSet{}
	helps := map[string]*bool{}
	for _, cmd := range subcommands {
		flagSets[cmd] = flag.NewFlagSet(cmd, flag.ExitOnError)
		val := false
		helps[cmd] = &val
		shorthandedFlag(flagSets[cmd].BoolVar, helps[cmd], "help", false, helpDoc)
	}
	var uncheckedOnly bool
	shorthandedFlag(
		flagSets["list"].BoolVar,
		&uncheckedOnly,
		"unchecked-only",
		false,
		"Only display the tasks that are not checked",
	)
	parseAll(flagSets)

	if mainHelp {
		moduleDoc()
	}
	if len(os.Args) < 2 || !slices.Contains(subcommands,os.Args[1]) {
		fmt.Print("Expected one subcommands of `", strings.Join(subcommands, "`, `"), "`\n")
        os.Exit(1)
	}
	for cmd, cmdHelp := range helps {
		if *cmdHelp {
			subcommandDoc(flagSets[cmd])
		}
	}
	
	todos, err := createDB()
	if err != nil {
		fmt.Println("Error initializing DB")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		todos.execInsert(strings.Join(os.Args[2:], " "))
	case "list":
		todos.execList(uncheckedOnly)
	case "toggle":
		todos.execToggleTask()
	case "remove":
	case "help":
		moduleDoc()
	default:
		fmt.Println("This branch should be unreachable...")
		os.Exit(1)
	}
}

func shorthandedFlag[T bool](
	fun func(*T, string, T, string),
	v *T,
	name string,
	defaultV T,
	description string,
) {
	fun(v, name, defaultV, description)
	fun(v, string(name[0]), defaultV, fmt.Sprint("Shorthand for ```", name, "`"))
}

func parseAll(flagSets map[string]*flag.FlagSet) string {
	flag.Parse()
	var subcommand string
	for cmd := range flagSets {
		if slices.Contains(os.Args[1:], cmd) {
			subcommand = cmd
			break
		}
	}
	if subcommand == "" {
		return ""
	}
	var subcommandStartIndex int
	for i := range os.Args {
		if os.Args[i] == subcommand {
			subcommandStartIndex = i
			break
		}
	}
	flagSets[subcommand].Parse(os.Args[subcommandStartIndex+1:])
	return subcommand
}


func moduleDoc() {
	fmt.Println("CLI tool to manage your tasks.\nList of available subcommands:")
	maxCmdLen := 0
	for _, cmd := range subcommands {
		if len(cmd) > maxCmdLen {
			maxCmdLen = len(cmd)
		}
	}
	for _, cmd := range subcommands {
		fmt.Printf("  %s%*s : %s\n", cmd, maxCmdLen-len(cmd), "", subcommandsDoc[cmd])
	}
	fmt.Println("\nList of available option:")
	flag.PrintDefaults()
	os.Exit(0)
}

func subcommandDoc(flagSet *flag.FlagSet) {
	cmd := flagSet.Name()
	fmt.Println(subcommandsDoc[cmd])
	if flagSet.NFlag() != 0 {
		fmt.Println("List of available options:")
		flagSet.PrintDefaults()
	}
	os.Exit(0)
}