package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devmarvs/bebo/migrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "new":
		newCmd(os.Args[2:])
	case "route":
		routeCmd(os.Args[2:])
	case "migrate":
		migrateCmd(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Println("bebo CLI")
	fmt.Println("\nCommands:")
	fmt.Println("  bebo new <dir> -module <module> [-version v0.0.0] [-template]")
	fmt.Println("  bebo route add -method GET -path /users/:id [-name user.show]")
	fmt.Println("  bebo migrate new -dir ./migrations -name create_users")
	fmt.Println("  bebo migrate plan -dir ./migrations [-driver postgres -dsn <dsn>]")
	fmt.Println("  bebo migrate up -dir ./migrations -driver postgres -dsn <dsn> [-lock-id 0]")
	fmt.Println("  bebo migrate down -dir ./migrations -driver postgres -dsn <dsn> -steps 1 [-lock-id 0]")
}

func newCmd(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	module := fs.String("module", "", "Go module path (required)")
	version := fs.String("version", "v0.0.0", "bebo version for go.mod")
	template := fs.Bool("template", false, "include templates")
	_ = fs.Parse(args)

	if fs.NArg() < 1 || *module == "" {
		fmt.Println("usage: bebo new <dir> -module <module> [-version v0.0.0] [-template]")
		return
	}

	dir := fs.Arg(0)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err)
	}

	if err := writeFile(filepath.Join(dir, "go.mod"), goMod(*module, *version)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(dir, "main.go"), mainGo(*module)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(dir, "README.md"), readme(*module)); err != nil {
		fatal(err)
	}

	if *template {
		tmplDir := filepath.Join(dir, "templates")
		if err := os.MkdirAll(tmplDir, 0o755); err != nil {
			fatal(err)
		}
		if err := writeFile(filepath.Join(tmplDir, "layout.html"), layoutTemplate()); err != nil {
			fatal(err)
		}
		if err := writeFile(filepath.Join(tmplDir, "home.html"), homeTemplate()); err != nil {
			fatal(err)
		}
	}

	fmt.Println("project created at", dir)
}

func routeCmd(args []string) {
	if len(args) == 0 || args[0] != "add" {
		fmt.Println("usage: bebo route add -method GET -path /users/:id [-name user.show]")
		return
	}

	fs := flag.NewFlagSet("route add", flag.ExitOnError)
	method := fs.String("method", "GET", "HTTP method")
	path := fs.String("path", "/", "Route path")
	name := fs.String("name", "", "Route name")
	_ = fs.Parse(args[1:])

	line := fmt.Sprintf("app.%s(%q, handler)", strings.ToUpper(*method), *path)
	if *name != "" {
		line = fmt.Sprintf("app.Route(%q, %q, handler, bebo.WithName(%q))", strings.ToUpper(*method), *path, *name)
	}

	fmt.Println(line)
}

func migrateCmd(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: bebo migrate <new|plan|up|down> ...")
		return
	}

	switch args[0] {
	case "new":
		migrateNewCmd(args[1:])
	case "plan":
		migratePlanCmd(args[1:])
	case "up":
		migrateUpCmd(args[1:])
	case "down":
		migrateDownCmd(args[1:])
	default:
		fmt.Println("usage: bebo migrate <new|plan|up|down> ...")
	}
}

func migrateNewCmd(args []string) {
	fs := flag.NewFlagSet("migrate new", flag.ExitOnError)
	dir := fs.String("dir", "migrations", "Migrations directory")
	name := fs.String("name", "", "Migration name")
	_ = fs.Parse(args)

	if *name == "" {
		fmt.Println("usage: bebo migrate new -dir ./migrations -name create_users")
		return
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fatal(err)
	}

	version := time.Now().UTC().Format("20060102150405")
	slug := sanitizeName(*name)
	base := fmt.Sprintf("%s_%s", version, slug)

	upPath := filepath.Join(*dir, base+".up.sql")
	downPath := filepath.Join(*dir, base+".down.sql")

	if err := writeFile(upPath, "-- write migration here\n"); err != nil {
		fatal(err)
	}
	if err := writeFile(downPath, "-- rollback migration here\n"); err != nil {
		fatal(err)
	}

	fmt.Println("created", upPath)
	fmt.Println("created", downPath)
}

func migratePlanCmd(args []string) {
	fs := flag.NewFlagSet("migrate plan", flag.ExitOnError)
	dir := fs.String("dir", "migrations", "Migrations directory")
	driver := fs.String("driver", "", "Database driver")
	dsn := fs.String("dsn", "", "Database DSN")
	_ = fs.Parse(args)

	var db *sql.DB
	var err error
	if *driver != "" && *dsn != "" {
		db, err = sql.Open(*driver, *dsn)
		if err != nil {
			fatal(err)
		}
		defer db.Close()
	}

	runner := migrate.New(db, *dir)
	plan, err := runner.Plan(context.Background())
	if err != nil {
		fatal(err)
	}

	for _, entry := range plan {
		status := "pending"
		if entry.Applied {
			status = "applied"
		}
		fmt.Printf("%s %d %s\n", status, entry.Version, entry.Name)
	}
}

func migrateUpCmd(args []string) {
	fs := flag.NewFlagSet("migrate up", flag.ExitOnError)
	dir := fs.String("dir", "migrations", "Migrations directory")
	driver := fs.String("driver", "", "Database driver")
	dsn := fs.String("dsn", "", "Database DSN")
	lockID := fs.Int64("lock-id", 0, "Postgres advisory lock ID")
	_ = fs.Parse(args)

	if *driver == "" || *dsn == "" {
		fmt.Println("usage: bebo migrate up -dir ./migrations -driver postgres -dsn <dsn> [-lock-id 0]")
		return
	}

	db, err := sql.Open(*driver, *dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	runner := migrate.New(db, *dir)
	if *lockID != 0 {
		runner.Locker = migrate.AdvisoryLocker{ID: *lockID}
	}

	count, err := runner.Up(context.Background())
	if err != nil {
		fatal(err)
	}
	fmt.Printf("applied %d migrations\n", count)
}

func migrateDownCmd(args []string) {
	fs := flag.NewFlagSet("migrate down", flag.ExitOnError)
	dir := fs.String("dir", "migrations", "Migrations directory")
	driver := fs.String("driver", "", "Database driver")
	dsn := fs.String("dsn", "", "Database DSN")
	steps := fs.Int("steps", 1, "Number of migrations to rollback")
	lockID := fs.Int64("lock-id", 0, "Postgres advisory lock ID")
	_ = fs.Parse(args)

	if *driver == "" || *dsn == "" {
		fmt.Println("usage: bebo migrate down -dir ./migrations -driver postgres -dsn <dsn> -steps 1 [-lock-id 0]")
		return
	}

	db, err := sql.Open(*driver, *dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	runner := migrate.New(db, *dir)
	if *lockID != 0 {
		runner.Locker = migrate.AdvisoryLocker{ID: *lockID}
	}

	count, err := runner.Down(context.Background(), *steps)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("rolled back %d migrations\n", count)
}

func goMod(module, version string) string {
	return fmt.Sprintf("module %s\n\ngo 1.25\n\nrequire github.com/devmarvs/bebo %s\n", module, version)
}

func mainGo(module string) string {
	_ = module
	return `package main

import (
	"log"
	"net/http"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/middleware"
)

func main() {
	app := bebo.New()
	app.Use(middleware.RequestID(), middleware.Recover(), middleware.Logger())

	app.GET("/health", func(ctx *bebo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	if err := app.RunWithSignals(); err != nil {
		log.Fatal(err)
	}
}
`
}

func readme(module string) string {
	return fmt.Sprintf("# %s\n\nGenerated by bebo.\n", module)
}

func layoutTemplate() string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
</head>
<body>
  {{ template "content" . }}
</body>
</html>
`
}

func homeTemplate() string {
	return `{{ define "content" }}
<h1>{{ .Title }}</h1>
{{ end }}
`
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^a-z0-9_]+`)
	name = strings.ReplaceAll(name, " ", "_")
	name = re.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		name = "migration"
	}
	return name
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
