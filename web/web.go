package web

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"github.com/gorilla/mux"
	"henrymail/config"
	"henrymail/database"
	"henrymail/model"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

func NewView(layout string, files ...string) *View {
	files = append(files,
		"web/templates/index.html",
		"web/templates/navigation.html",
		"web/templates/header.html",
		"web/templates/footer.html",
	)
	t, err := template.ParseFiles(files...)
	if err != nil {
		log.Fatal(err)
	}
	return &View{
		Template: t,
		Layout:   layout,
	}
}

type View struct {
	Template *template.Template
	Layout   string
}

func (v *View) Render(w http.ResponseWriter, viewModel interface{}) {
	e := v.Template.ExecuteTemplate(w, v.Layout, viewModel)
	if e != nil {
		log.Print(e)
	}
}

// Data required for header, navigation etc
type LayoutData struct {
	Mailboxes   []*model.Mbx
	CurrentUser *model.Usr
}

type wa struct {
	lg        database.Login
	db        database.Database
	jwtSecret []byte
	pk        *rsa.PublicKey

	loginView          *View
	mailboxView        *View
	errorView          *View
	changePasswordView *View
	messageView        *View
	usersView          *View
	healthChecksView   *View
}

type AuthenticatedHandler = func(w http.ResponseWriter, r *http.Request, u *model.Usr)

func (wa *wa) renderError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	wa.errorView.Render(w, err)
}

func (wa *wa) root(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	http.Redirect(w, r, "/mailbox/INBOX", http.StatusFound)
}

func (wa *wa) layoutData(u *model.Usr) (*LayoutData, error) {
	mbxes, e := wa.db.GetMailboxes(true, u.Id)
	if e != nil {
		return nil, e
	}
	return &LayoutData{
		CurrentUser: u,
		Mailboxes:   mbxes,
	}, nil
}

func (wa *wa) msgs(name string, u *model.Usr) (*model.Mbx, []*model.Msg, error) {
	mbx, e := wa.db.GetMailboxByName(name, u.Id)
	if e != nil {
		return nil, nil, e
	}
	msgs, e := wa.db.GetMessages(mbx.Id, -1, -1)
	if e != nil {
		return nil, nil, e
	}
	return mbx, msgs, nil
}

func StartWebAdmin(lg database.Login, db database.Database, tlsC *tls.Config, pk *rsa.PublicKey) {
	// Generate or read secret for JWT auth
	jwtSecret, e := ioutil.ReadFile(config.GetString(config.JwtTokenSecretFile))
	if os.IsNotExist(e) {
		jwtSecret = generateAndSaveJwtSecret()
	} else if e != nil {
		log.Fatal(e)
	}

	// Read the templates
	tp := template.New("html")
	e = filepath.Walk("web/templates", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			_, err = tp.ParseFiles(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if e != nil {
		log.Fatal(e)
	}

	webAdmin := wa{
		lg:                 lg,
		db:                 db,
		jwtSecret:          jwtSecret,
		pk:                 pk,
		loginView:          NewView("login.html", "web/templates/login.html"),
		mailboxView:        NewView("index.html", "web/templates/mailbox.html"),
		changePasswordView: NewView("index.html", "web/templates/change_password.html"),
		messageView:        NewView("index.html", "web/templates/message.html"),
		usersView:          NewView("index.html", "web/templates/users.html"),
		healthChecksView:   NewView("index.html", "web/templates/health_checks.html"),
	}

	if e != nil {
		log.Fatal(e)
	}
	router := mux.NewRouter()
	router.HandleFunc("/login", webAdmin.login)
	router.HandleFunc("/logout", webAdmin.logout)
	router.Handle("/changePassword", webAdmin.checkLogin(webAdmin.changePassword))
	router.Handle("/mailbox/{name}", webAdmin.checkLogin(webAdmin.mailbox))
	router.Handle("/mailbox/{name}/{id}", webAdmin.checkLogin(webAdmin.message))
	router.Handle("/", webAdmin.checkLogin(webAdmin.root))

	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	admin := router.PathPrefix("/admin/").Subrouter()
	admin.Handle("/users", webAdmin.checkAdmin(webAdmin.users))
	admin.Handle("/addUser", webAdmin.checkAdmin(webAdmin.add))
	admin.Handle("/deleteUser", webAdmin.checkAdmin(webAdmin.delete))
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

func generateAndSaveJwtSecret() []byte {
	jwtSecret := make([]byte, 64)
	_, e := rand.Read(jwtSecret)
	if e != nil {
		log.Fatal(e)
	}
	e = ioutil.WriteFile(config.GetString(config.JwtTokenSecretFile), jwtSecret, 0700)
	if e != nil {
		log.Fatal(e)
	}
	return jwtSecret
}
