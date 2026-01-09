package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/config"
	"github.com/devmarvs/bebo/flash"
	"github.com/devmarvs/bebo/health"
	"github.com/devmarvs/bebo/middleware"
	"github.com/devmarvs/bebo/session"
	"github.com/devmarvs/bebo/validate"
	"github.com/devmarvs/bebo/web"
)

const userKey = "currentUser"

type AppConfig struct {
	App           config.Config
	DatabaseURL   string
	SessionKey    []byte
	SecureCookies bool
	AutoMigrate   bool
}

type Server struct {
	store    *Store
	sessions session.Store
	flash    flash.Store
}

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Note struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type viewData struct {
	Title string
	User  *User
	Notes []Note
	Note  *Note
	Email string
	Error string
}

type signupForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
}

type loginForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required"`
}

type noteForm struct {
	Title string `form:"title" validate:"required,min=2"`
	Body  string `form:"body" validate:"required,min=2"`
}

type notePayload struct {
	Title string `json:"title" validate:"required,min=2"`
	Body  string `json:"body" validate:"required,min=2"`
}

func NewApp(dbConn *sql.DB, cfg AppConfig) *bebo.App {
	appCfg := cfg.App
	if appCfg.TemplatesDir == "" {
		appCfg.TemplatesDir = templatesDir()
	}
	if appCfg.LayoutTemplate == "" {
		appCfg.LayoutTemplate = "layout.html"
	}

	app := bebo.New(
		bebo.WithConfig(appCfg),
		bebo.WithTemplateFuncs(web.Funcs()),
		bebo.WithTemplatePartials("partials/*.html"),
	)

	app.Use(
		middleware.RequestID(),
		middleware.Recover(),
		middleware.Logger(),
		middleware.SecurityHeaders(middleware.DefaultSecurityHeaders()),
		middleware.CSRF(middleware.CSRFOptions{CookieSecure: cfg.SecureCookies}),
	)
	app.UsePre(middleware.MethodOverride(middleware.MethodOverrideOptions{}))

	cookieStore := session.NewCookieStore("bebo_session", cfg.SessionKey)
	cookieStore.Secure = cfg.SecureCookies
	cookieStore.HTTPOnly = true

	server := &Server{
		store:    NewStore(dbConn),
		sessions: cookieStore,
		flash:    flash.New(cookieStore),
	}

	registry := health.New(health.WithTimeout(2 * time.Second))
	registry.Add("db", func(ctx context.Context) error {
		return dbConn.PingContext(ctx)
	})
	registry.AddReady("db", func(ctx context.Context) error {
		return dbConn.PingContext(ctx)
	})

	app.GET("/health", func(ctx *bebo.Context) error {
		registry.Handler().ServeHTTP(ctx.ResponseWriter, ctx.Request)
		return nil
	})
	app.GET("/ready", func(ctx *bebo.Context) error {
		registry.ReadyHandler().ServeHTTP(ctx.ResponseWriter, ctx.Request)
		return nil
	})

	app.GET("/", server.home)
	app.GET("/signup", server.signupForm)
	app.POST("/signup", server.signup)
	app.GET("/login", server.loginForm)
	app.POST("/login", server.login)
	app.POST("/logout", server.logout)

	notes := app.Group("/notes", server.requireUser(true))
	notes.GET("", server.notesIndex)
	notes.GET("/new", server.notesNew)
	notes.POST("", server.notesCreate)
	notes.GET("/:id", server.notesShow)
	notes.GET("/:id/edit", server.notesEdit)
	notes.PUT("/:id", server.notesUpdate)
	notes.DELETE("/:id", server.notesDelete)

	api := app.Group("/api", server.requireUser(false))
	api.GET("/notes", server.apiNotesIndex)
	api.GET("/notes/:id", server.apiNotesShow)
	api.POST("/notes", server.apiNotesCreate)
	api.PUT("/notes/:id", server.apiNotesUpdate)
	api.DELETE("/notes/:id", server.apiNotesDelete)

	return app
}

func templatesDir() string {
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	return filepath.Join("examples", "crud", "templates")
}

func userFromContext(ctx *bebo.Context) (*User, bool) {
	value, ok := ctx.Get(userKey)
	if !ok {
		return nil, false
	}
	user, ok := value.(*User)
	return user, ok
}

func mustUser(ctx *bebo.Context) (*User, error) {
	user, ok := userFromContext(ctx)
	if !ok || user == nil {
		return nil, apperr.Unauthorized("login required", nil)
	}
	return user, nil
}

func (s *Server) currentUser(ctx *bebo.Context) (*User, error) {
	sess, err := s.sessions.Get(ctx.Request)
	if err != nil {
		return nil, err
	}
	idValue := sess.Get("user_id")
	if idValue == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(idValue, 10, 64)
	if err != nil {
		sess.Delete("user_id")
		_ = sess.Save(ctx.ResponseWriter)
		return nil, nil
	}
	user, err := s.store.UserByID(ctx.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		sess.Delete("user_id")
		_ = sess.Save(ctx.ResponseWriter)
		return nil, nil
	}
	return user, err
}

func (s *Server) signIn(ctx *bebo.Context, user *User) error {
	sess, err := s.sessions.Get(ctx.Request)
	if err != nil {
		return err
	}
	sess.Set("user_id", strconv.FormatInt(user.ID, 10))
	return sess.Save(ctx.ResponseWriter)
}

func (s *Server) requireUser(redirectToLogin bool) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			user, err := s.currentUser(ctx)
			if err != nil {
				return err
			}
			if user == nil {
				if redirectToLogin {
					_ = s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
						Type: "error",
						Text: "Please log in to continue.",
					})
					return redirect(ctx, "/login")
				}
				return apperr.Unauthorized("login required", nil)
			}
			ctx.Set(userKey, user)
			return next(ctx)
		}
	}
}

func (s *Server) render(ctx *bebo.Context, status int, name string, data viewData) error {
	if data.User == nil {
		if user, ok := userFromContext(ctx); ok {
			data.User = user
		}
	}
	view, err := web.TemplateDataFrom(ctx, &s.flash, data)
	if err != nil {
		return err
	}
	return ctx.HTML(status, name, view)
}

func (s *Server) home(ctx *bebo.Context) error {
	user, err := s.currentUser(ctx)
	if err != nil {
		return err
	}
	if user == nil {
		return redirect(ctx, "/login")
	}
	return redirect(ctx, "/notes")
}

func (s *Server) signupForm(ctx *bebo.Context) error {
	if user, err := s.currentUser(ctx); err != nil {
		return err
	} else if user != nil {
		return redirect(ctx, "/notes")
	}
	return s.render(ctx, http.StatusOK, "auth/signup.html", viewData{Title: "Sign up"})
}

func (s *Server) signup(ctx *bebo.Context) error {
	var form signupForm
	if err := ctx.BindForm(&form); err != nil {
		return err
	}
	if err := validate.Struct(form); err != nil {
		return s.render(ctx, http.StatusBadRequest, "auth/signup.html", viewData{
			Title: "Sign up",
			Email: form.Email,
			Error: validationMessage(err),
		})
	}

	user, err := s.store.CreateUser(ctx.Request.Context(), form.Email, form.Password)
	if errors.Is(err, ErrConflict) {
		return s.render(ctx, http.StatusBadRequest, "auth/signup.html", viewData{
			Title: "Sign up",
			Email: form.Email,
			Error: "Email already exists.",
		})
	}
	if err != nil {
		return err
	}

	if err := s.signIn(ctx, user); err != nil {
		return err
	}
	if err := s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
		Type: "success",
		Text: "Account created.",
	}); err != nil {
		return err
	}
	return redirect(ctx, "/notes")
}

func (s *Server) loginForm(ctx *bebo.Context) error {
	if user, err := s.currentUser(ctx); err != nil {
		return err
	} else if user != nil {
		return redirect(ctx, "/notes")
	}
	return s.render(ctx, http.StatusOK, "auth/login.html", viewData{Title: "Log in"})
}

func (s *Server) login(ctx *bebo.Context) error {
	var form loginForm
	if err := ctx.BindForm(&form); err != nil {
		return err
	}
	if err := validate.Struct(form); err != nil {
		return s.render(ctx, http.StatusBadRequest, "auth/login.html", viewData{
			Title: "Log in",
			Email: form.Email,
			Error: validationMessage(err),
		})
	}

	user, err := s.store.Authenticate(ctx.Request.Context(), form.Email, form.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return s.render(ctx, http.StatusUnauthorized, "auth/login.html", viewData{
			Title: "Log in",
			Email: form.Email,
			Error: "Invalid email or password.",
		})
	}
	if err != nil {
		return err
	}

	if err := s.signIn(ctx, user); err != nil {
		return err
	}
	if err := s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
		Type: "success",
		Text: "Welcome back.",
	}); err != nil {
		return err
	}
	return redirect(ctx, "/notes")
}

func (s *Server) logout(ctx *bebo.Context) error {
	sess, err := s.sessions.Get(ctx.Request)
	if err != nil {
		return err
	}
	sess.Clear(ctx.ResponseWriter)
	if err := s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
		Type: "success",
		Text: "Signed out.",
	}); err != nil {
		return err
	}
	return redirect(ctx, "/login")
}

func (s *Server) notesIndex(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	notes, err := s.store.ListNotes(ctx.Request.Context(), user.ID)
	if err != nil {
		return err
	}
	return s.render(ctx, http.StatusOK, "notes/index.html", viewData{
		Title: "Your notes",
		User:  user,
		Notes: notes,
	})
}

func (s *Server) notesNew(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	return s.render(ctx, http.StatusOK, "notes/new.html", viewData{
		Title: "New note",
		User:  user,
	})
}

func (s *Server) notesCreate(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}

	var form noteForm
	if err := ctx.BindForm(&form); err != nil {
		return err
	}
	if err := validate.Struct(form); err != nil {
		return s.render(ctx, http.StatusBadRequest, "notes/new.html", viewData{
			Title: "New note",
			User:  user,
			Error: validationMessage(err),
		})
	}

	note, err := s.store.CreateNote(ctx.Request.Context(), user.ID, form.Title, form.Body)
	if err != nil {
		return err
	}
	if err := s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
		Type: "success",
		Text: "Note created.",
	}); err != nil {
		return err
	}
	return redirect(ctx, "/notes/"+strconv.FormatInt(note.ID, 10))
}

func (s *Server) notesShow(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}

	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	note, err := s.store.NoteByID(ctx.Request.Context(), user.ID, noteID)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	}
	if err != nil {
		return err
	}

	return s.render(ctx, http.StatusOK, "notes/show.html", viewData{
		Title: "View note",
		User:  user,
		Note:  note,
	})
}

func (s *Server) notesEdit(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}

	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	note, err := s.store.NoteByID(ctx.Request.Context(), user.ID, noteID)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	}
	if err != nil {
		return err
	}

	return s.render(ctx, http.StatusOK, "notes/edit.html", viewData{
		Title: "Edit note",
		User:  user,
		Note:  note,
	})
}

func (s *Server) notesUpdate(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}

	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	var form noteForm
	if err := ctx.BindForm(&form); err != nil {
		return err
	}
	if err := validate.Struct(form); err != nil {
		return s.render(ctx, http.StatusBadRequest, "notes/edit.html", viewData{
			Title: "Edit note",
			User:  user,
			Note:  &Note{ID: noteID, Title: form.Title, Body: form.Body},
			Error: validationMessage(err),
		})
	}

	note, err := s.store.UpdateNote(ctx.Request.Context(), user.ID, noteID, form.Title, form.Body)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	}
	if err != nil {
		return err
	}
	if err := s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
		Type: "success",
		Text: "Note updated.",
	}); err != nil {
		return err
	}
	return redirect(ctx, "/notes/"+strconv.FormatInt(note.ID, 10))
}

func (s *Server) notesDelete(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}

	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	if err := s.store.DeleteNote(ctx.Request.Context(), user.ID, noteID); errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	} else if err != nil {
		return err
	}
	if err := s.flash.Add(ctx.ResponseWriter, ctx.Request, flash.Message{
		Type: "success",
		Text: "Note deleted.",
	}); err != nil {
		return err
	}
	return redirect(ctx, "/notes")
}

func (s *Server) apiNotesIndex(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	notes, err := s.store.ListNotes(ctx.Request.Context(), user.ID)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, notes)
}

func (s *Server) apiNotesShow(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	note, err := s.store.NoteByID(ctx.Request.Context(), user.ID, noteID)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	}
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, note)
}

func (s *Server) apiNotesCreate(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	var payload notePayload
	if err := ctx.BindJSON(&payload); err != nil {
		return err
	}
	if err := validate.Struct(payload); err != nil {
		return err
	}
	note, err := s.store.CreateNote(ctx.Request.Context(), user.ID, payload.Title, payload.Body)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, note)
}

func (s *Server) apiNotesUpdate(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	var payload notePayload
	if err := ctx.BindJSON(&payload); err != nil {
		return err
	}
	if err := validate.Struct(payload); err != nil {
		return err
	}
	note, err := s.store.UpdateNote(ctx.Request.Context(), user.ID, noteID, payload.Title, payload.Body)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	}
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, note)
}

func (s *Server) apiNotesDelete(ctx *bebo.Context) error {
	user, err := mustUser(ctx)
	if err != nil {
		return err
	}
	noteID, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	if err := s.store.DeleteNote(ctx.Request.Context(), user.ID, noteID); errors.Is(err, ErrNotFound) {
		return apperr.NotFound("note not found", err)
	} else if err != nil {
		return err
	}
	ctx.ResponseWriter.WriteHeader(http.StatusNoContent)
	return nil
}

func validationMessage(err error) string {
	if verr, ok := validate.As(err); ok && len(verr.Fields) > 0 {
		return verr.Fields[0].Message
	}
	if err != nil {
		return err.Error()
	}
	return "validation failed"
}

func redirect(ctx *bebo.Context, url string) error {
	http.Redirect(ctx.ResponseWriter, ctx.Request, url, http.StatusSeeOther)
	return nil
}
