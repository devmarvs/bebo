package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

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
	case "crud":
		crudCmd(os.Args[2:])
	case "migrate":
		migrateCmd(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Println("bebo CLI")
	fmt.Println("\nCommands:")
	fmt.Println("  bebo new <dir> -module <module> [-version v0.0.0] [-template] [-profile]")
	fmt.Println("  bebo route add -method GET -path /users/:id [-name user.show]")
	fmt.Println("  bebo crud new <resource> [-dir handlers] [-package handlers] [-templates templates] [-tests=true]")
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
	profile := fs.Bool("profile", false, "include config profiles")
	_ = fs.Parse(args)

	if fs.NArg() < 1 || *module == "" {
		fmt.Println("usage: bebo new <dir> -module <module> [-version v0.0.0] [-template] [-profile]")
		return
	}

	dir := fs.Arg(0)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err)
	}

	if err := writeFile(filepath.Join(dir, "go.mod"), goMod(*module, *version)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(dir, "main.go"), mainGo(*module, *profile)); err != nil {
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

	if *profile {
		configDir := filepath.Join(dir, "config")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			fatal(err)
		}
		if err := writeFile(filepath.Join(configDir, "base.json"), configBaseTemplate()); err != nil {
			fatal(err)
		}
		if err := writeFile(filepath.Join(configDir, "development.json"), configEnvTemplate()); err != nil {
			fatal(err)
		}
		if err := writeFile(filepath.Join(configDir, "secrets.example.json"), configSecretsTemplate()); err != nil {
			fatal(err)
		}
		_ = writeFileIfNotExists(filepath.Join(dir, ".gitignore"), gitignoreTemplate())
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

func crudCmd(args []string) {
	if len(args) == 0 || args[0] != "new" {
		fmt.Println("usage: bebo crud new <resource> [-dir handlers] [-package handlers] [-templates templates] [-tests=true]")
		return
	}

	crudNewCmd(args[1:])
}

func crudNewCmd(args []string) {
	fs := flag.NewFlagSet("crud new", flag.ExitOnError)
	dir := fs.String("dir", "handlers", "Output directory")
	pkg := fs.String("package", "", "Go package name (default: base of dir)")
	templatesDir := fs.String("templates", "templates", "Templates root directory (empty to skip)")
	tests := fs.Bool("tests", true, "Generate tests")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("usage: bebo crud new <resource> [-dir handlers] [-package handlers] [-templates templates] [-tests=true]")
		return
	}

	resource := sanitizeResourceName(fs.Arg(0))
	if resource == "" {
		fmt.Println("resource name is required")
		return
	}

	singular, plural := resourceNames(resource)
	if singular == "" || plural == "" {
		fmt.Println("invalid resource name")
		return
	}

	if *pkg == "" {
		*pkg = packageNameFromDir(*dir)
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fatal(err)
	}

	handlerPath := filepath.Join(*dir, plural+".go")
	if err := writeFileIfNotExists(handlerPath, crudHandlerTemplate(*pkg, singular, plural)); err != nil {
		fatal(err)
	}

	if *tests {
		testPath := filepath.Join(*dir, plural+"_test.go")
		if err := writeFileIfNotExists(testPath, crudTestTemplate(*pkg, singular, plural)); err != nil {
			fatal(err)
		}
	}

	templatesRoot := strings.TrimSpace(*templatesDir)
	if templatesRoot != "" {
		resourceDir := filepath.Join(templatesRoot, plural)
		if err := os.MkdirAll(resourceDir, 0o755); err != nil {
			fatal(err)
		}
		if err := writeFileIfNotExists(filepath.Join(resourceDir, "index.html"), crudIndexTemplate(singular, plural)); err != nil {
			fatal(err)
		}
		if err := writeFileIfNotExists(filepath.Join(resourceDir, "show.html"), crudShowTemplate(singular, plural)); err != nil {
			fatal(err)
		}
		if err := writeFileIfNotExists(filepath.Join(resourceDir, "new.html"), crudNewTemplate(singular, plural)); err != nil {
			fatal(err)
		}
		if err := writeFileIfNotExists(filepath.Join(resourceDir, "edit.html"), crudEditTemplate(singular, plural)); err != nil {
			fatal(err)
		}
	}

	fmt.Println("crud files created:")
	fmt.Println("  " + handlerPath)
	if *tests {
		fmt.Println("  " + filepath.Join(*dir, plural+"_test.go"))
	}
	if strings.TrimSpace(*templatesDir) != "" {
		fmt.Println("  " + filepath.Join(*templatesDir, plural, "index.html"))
		fmt.Println("  " + filepath.Join(*templatesDir, plural, "show.html"))
		fmt.Println("  " + filepath.Join(*templatesDir, plural, "new.html"))
		fmt.Println("  " + filepath.Join(*templatesDir, plural, "edit.html"))
	}
	fmt.Println("register routes:")
	fmt.Printf("  %s.Register%sRoutes(app)\n", *pkg, pascalCase(singular))
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

func mainGo(module string, withProfile bool) string {
	_ = module
	if withProfile {
		return `package main

import (
	"log"
	"net/http"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/config"
	"github.com/devmarvs/bebo/middleware"
)

func main() {
	profile := config.Profile{
		BasePath:    "config/base.json",
		EnvPath:     "config/development.json",
		SecretsPath: "config/secrets.json",
		EnvPrefix:   "BEBO_",
		AllowMissing: true,
	}
	cfg, err := config.LoadProfile(profile)
	if err != nil {
		log.Fatal(err)
	}

	app := bebo.New(bebo.WithConfig(cfg))
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
	return fmt.Sprintf("# %s\n", module)
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


func configBaseTemplate() string {
	return `{
  "Address": ":8080",
  "ReadTimeout": "10s",
  "WriteTimeout": "20s",
  "IdleTimeout": "60s",
  "ReadHeaderTimeout": "5s",
  "ShutdownTimeout": "10s",
  "MaxHeaderBytes": 1048576,
  "TemplatesDir": "templates",
  "LayoutTemplate": "layout.html",
  "TemplateReload": false,
  "LogLevel": "info",
  "LogFormat": "text"
}
`
}

func configEnvTemplate() string {
	return `{
  "TemplateReload": true,
  "LogLevel": "debug"
}
`
}

func configSecretsTemplate() string {
	return `{}
`
}

func gitignoreTemplate() string {
	return "config/secrets.json\n"
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}

func writeFileIfNotExists(path, contents string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeFile(path, contents)
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

func sanitizeResourceName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^a-z0-9_]+`)
	name = strings.ReplaceAll(name, " ", "_")
	name = re.ReplaceAllString(name, "_")
	return strings.Trim(name, "_")
}

func packageNameFromDir(dir string) string {
	base := filepath.Base(filepath.Clean(dir))
	name := sanitizeResourceName(base)
	if name == "" {
		return "handlers"
	}
	return name
}

func resourceNames(name string) (string, string) {
	if looksPlural(name) {
		return singularize(name), name
	}
	return name, pluralize(name)
}

func looksPlural(name string) bool {
	name = strings.ToLower(name)
	if strings.HasSuffix(name, "ss") || strings.HasSuffix(name, "us") {
		return false
	}
	return strings.HasSuffix(name, "s")
}

func pluralize(name string) string {
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "z") ||
		strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		return name + "es"
	}
	if strings.HasSuffix(lower, "y") && len(lower) > 1 && !isVowel(lower[len(lower)-2]) {
		return name[:len(name)-1] + "ies"
	}
	return name + "s"
}

func singularize(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "ies") && len(name) > 3:
		return name[:len(name)-3] + "y"
	case strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "shes") ||
		strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") ||
		strings.HasSuffix(lower, "zes"):
		return name[:len(name)-2]
	case strings.HasSuffix(lower, "s") && len(name) > 1:
		return name[:len(name)-1]
	default:
		return name
	}
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func titleCase(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return name
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		out = append(out, string(runes))
	}
	return strings.Join(out, " ")
}

func pascalCase(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return name
	}
	var b strings.Builder
	for _, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

func crudHandlerTemplate(pkg, singular, plural string) string {
	typeName := pascalCase(singular)
	pageType := pascalCase(singular) + "PageData"
	registerName := pascalCase(singular)

	listFn := "list" + pascalCase(plural)
	showFn := "show" + pascalCase(singular)
	newFn := "new" + pascalCase(singular)
	editFn := "edit" + pascalCase(singular)
	createFn := "create" + pascalCase(singular)
	updateFn := "update" + pascalCase(singular)
	deleteFn := "delete" + pascalCase(singular)

	titlePlural := titleCase(plural)
	titleSingular := titleCase(singular)
	titleNew := "New " + titleSingular
	titleEdit := "Edit " + titleSingular

	return fmt.Sprintf(`package %s

import (
	"net/http"
	"strings"

	"github.com/devmarvs/bebo"
)

type %s struct {
	ID string `+"`json:\"id\"`"+`
}

type %s struct {
	Title string
	Item  %s
	Items []%s
}

func Register%sRoutes(app *bebo.App) {
	app.GET("/%s", %s)
	app.GET("/%s/new", %s)
	app.GET("/%s/:id", %s)
	app.GET("/%s/:id/edit", %s)
	app.POST("/%s", %s)
	app.PUT("/%s/:id", %s)
	app.DELETE("/%s/:id", %s)
}

func %s(ctx *bebo.Context) error {
	items := []%s{{ID: "1"}, {ID: "2"}}
	data := %s{
		Title: "%s",
		Items: items,
	}
	if acceptsHTML(ctx.Request) {
		if err := ctx.HTML(http.StatusOK, "%s/index.html", data); err == nil {
			return nil
		}
	}
	return ctx.JSON(http.StatusOK, items)
}

func %s(ctx *bebo.Context) error {
	id := ctx.Param("id")
	item := %s{ID: id}
	data := %s{
		Title: "%s",
		Item:  item,
	}
	if acceptsHTML(ctx.Request) {
		if err := ctx.HTML(http.StatusOK, "%s/show.html", data); err == nil {
			return nil
		}
	}
	return ctx.JSON(http.StatusOK, item)
}

func %s(ctx *bebo.Context) error {
	data := %s{
		Title: "%s",
	}
	if acceptsHTML(ctx.Request) {
		if err := ctx.HTML(http.StatusOK, "%s/new.html", data); err == nil {
			return nil
		}
	}
	return ctx.JSON(http.StatusOK, map[string]string{"status": "new"})
}

func %s(ctx *bebo.Context) error {
	id := ctx.Param("id")
	item := %s{ID: id}
	data := %s{
		Title: "%s",
		Item:  item,
	}
	if acceptsHTML(ctx.Request) {
		if err := ctx.HTML(http.StatusOK, "%s/edit.html", data); err == nil {
			return nil
		}
	}
	return ctx.JSON(http.StatusOK, item)
}

func %s(ctx *bebo.Context) error {
	return ctx.JSON(http.StatusCreated, map[string]string{"status": "created"})
}

func %s(ctx *bebo.Context) error {
	id := ctx.Param("id")
	return ctx.JSON(http.StatusOK, map[string]string{"status": "updated", "id": id})
}

func %s(ctx *bebo.Context) error {
	ctx.ResponseWriter.WriteHeader(http.StatusNoContent)
	return nil
}

func acceptsHTML(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/html")
}
`, pkg,
		typeName,
		pageType, typeName, typeName,
		registerName,
		plural, listFn,
		plural, newFn,
		plural, showFn,
		plural, editFn,
		plural, createFn,
		plural, updateFn,
		plural, deleteFn,
		listFn, typeName, pageType, titlePlural, plural,
		showFn, typeName, pageType, titleSingular, plural,
		newFn, pageType, titleNew, plural,
		editFn, typeName, pageType, titleEdit, plural,
		createFn,
		updateFn,
		deleteFn,
	)
}

func crudTestTemplate(pkg, singular, plural string) string {
	registerName := pascalCase(singular)

	return fmt.Sprintf(`package %s

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
)

func Test%sRoutes(t *testing.T) {
	app := bebo.New()
	Register%sRoutes(app)

	server := httptest.NewServer(app)
	defer server.Close()

	cases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"list", http.MethodGet, "/%s", http.StatusOK},
		{"new", http.MethodGet, "/%s/new", http.StatusOK},
		{"show", http.MethodGet, "/%s/1", http.StatusOK},
		{"edit", http.MethodGet, "/%s/1/edit", http.StatusOK},
		{"create", http.MethodPost, "/%s", http.StatusCreated},
		{"update", http.MethodPut, "/%s/1", http.StatusOK},
		{"delete", http.MethodDelete, "/%s/1", http.StatusNoContent},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, server.URL+tc.path, nil)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			req.Header.Set("Accept", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, resp.StatusCode)
			}
		})
	}
}
`, pkg, registerName, registerName, plural, plural, plural, plural, plural, plural, plural)
}

func crudIndexTemplate(singular, plural string) string {
	singularTitle := titleCase(singular)
	pluralTitle := titleCase(plural)

	return fmt.Sprintf(`{{ define "content" }}
<h1>{{ .Title }}</h1>
<ul>
  {{ range .Items }}
  <li><a href="/%s/{{ .ID }}">{{ .ID }}</a></li>
  {{ else }}
  <li>No %s yet.</li>
  {{ end }}
</ul>
<a href="/%s/new">New %s</a>
{{ end }}
`, plural, pluralTitle, plural, singularTitle)
}

func crudShowTemplate(singular, plural string) string {
	singularTitle := titleCase(singular)
	pluralTitle := titleCase(plural)

	return fmt.Sprintf(`{{ define "content" }}
<h1>{{ .Title }}</h1>
<p>ID: {{ .Item.ID }}</p>
<p><a href="/%s/{{ .Item.ID }}/edit">Edit %s</a></p>
<p><a href="/%s">Back to %s</a></p>
{{ end }}
`, plural, singularTitle, plural, pluralTitle)
}

func crudNewTemplate(singular, plural string) string {
	singularTitle := titleCase(singular)

	return fmt.Sprintf(`{{ define "content" }}
<h1>{{ .Title }}</h1>
<form method="post" action="/%s">
  <label>%s name <input name="name"></label>
  <button type="submit">Create</button>
</form>
{{ end }}
`, plural, singularTitle)
}

func crudEditTemplate(singular, plural string) string {
	singularTitle := titleCase(singular)

	return fmt.Sprintf(`{{ define "content" }}
<h1>{{ .Title }}</h1>
<form method="post" action="/%s/{{ .Item.ID }}">
  <input type="hidden" name="_method" value="PUT">
  <label>%s name <input name="name" value="{{ .Item.ID }}"></label>
  <button type="submit">Update</button>
</form>
<form method="post" action="/%s/{{ .Item.ID }}" style="margin-top:1rem;">
  <input type="hidden" name="_method" value="DELETE">
  <button type="submit">Delete</button>
</form>
<p><small>Note: add method override middleware or use JavaScript to send PUT/DELETE requests.</small></p>
{{ end }}
`, plural, singularTitle, plural)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
