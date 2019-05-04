package web

import (
	"crypto/tls"
	"database/sql"
	"github.com/gorilla/mux"
	"henrymail/config"
	"henrymail/models"
	"html/template"
	"log"
	"net"
	"net/http"
	"path"
)

//go:generate embed -c "embed.json"

// A page in the application
type view struct {
	tpl      *template.Template
	rootName string
}

// Data that's required to render header / navigation / footer
type layoutData struct {
	CurrentUser *models.User
}

type wa struct {
	db        *sql.DB
	jwtSecret []byte

	// All views are pre-loaded
	loginView          *view
	errorView          *view
	changePasswordView *view
	usersView          *view
	healthChecksView   *view
	securityView       *view
}

func newView(layout string, files ...string) *view {
	files = append(files,
		"/templates/index.html",
		"/templates/navigation.html",
		"/templates/header.html",
		"/templates/footer.html",
	)
	var t *template.Template = nil
	for _, file := range files {
		contents, e := GetEmbeddedContent().GetContents(file)
		if e != nil {
			log.Fatal(e)
		}
		name := path.Base(file)
		if t == nil {
			t = template.New(name)
		} else {
			t = t.New(name)
		}
		template.Must(t.Parse(string(contents)))
	}
	return &view{
		tpl:      t,
		rootName: layout,
	}
}

func (v *view) render(w http.ResponseWriter, viewModel interface{}) {
	e := v.tpl.ExecuteTemplate(w, v.rootName, viewModel)
	if e != nil {
		log.Print(e)
	}
}

func (wa *wa) renderError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	wa.errorView.render(w, err)
}

func (wa *wa) layoutData(u *models.User) (*layoutData, error) {
	return &layoutData{
		CurrentUser: u,
	}, nil
}

func StartWebAdmin(db *sql.DB, tlsC *tls.Config) {
	webAdmin := wa{
		db:                 db,
		jwtSecret:          loadJwtSecret(),
		loginView:          newView("login.html", "/templates/login.html"),
		changePasswordView: newView("index.html", "/templates/change_password.html"),
		usersView:          newView("index.html", "/templates/users.html"),
		healthChecksView:   newView("index.html", "/templates/healthchecks.html"),
		securityView:       newView("index.html", "/templates/security.html"),
		errorView:          newView("error.html", "/templates/error.html"),
	}

	router := mux.NewRouter()
	router.HandleFunc("/login", webAdmin.login)
	router.HandleFunc("/logout", webAdmin.logout)
	router.Handle("/", http.RedirectHandler("/changePassword", http.StatusTemporaryRedirect))
	router.Handle("/changePassword", webAdmin.checkLogin(webAdmin.changePassword))

	router.PathPrefix("/assets/").Handler(GetEmbeddedContent())

	admin := router.PathPrefix("/admin/").Subrouter()
	admin.Handle("/users", webAdmin.checkAdmin(webAdmin.users))
	admin.Handle("/addUser", webAdmin.checkAdmin(webAdmin.add))
	admin.Handle("/deleteUser", webAdmin.checkAdmin(webAdmin.delete))
	admin.Handle("/security", webAdmin.checkAdmin(webAdmin.security))
	admin.Handle("/rotateJwt", webAdmin.checkAdmin(webAdmin.rotateJwt))
	admin.Handle("/healthChecks", webAdmin.checkAdmin(webAdmin.healthChecks))

	server := &http.Server{Addr: config.GetString(config.WebAdminAddress), Handler: router}

	go func() {
		l, e := net.Listen("tcp", server.Addr)
		if e != nil {
			log.Fatal(e)
		}
		if config.GetBool(config.WebAdminUseTls) {
			l = tls.NewListener(l, tlsC)
		}
		log.Println("Started admin web server at ", server.Addr)
		if err := server.Serve(l); err != nil {
			log.Fatal(err)
		}
	}()
}
