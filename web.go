package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func NewView(layout string, files ...string) *View {
	files = append(files,
		"templates/index.html",
		"templates/navigation.html",
		"templates/header.html",
		"templates/footer.html",
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
	Mailboxes   []*Mbx
	CurrentUser *Usr
}

type wa struct {
	lg        Login
	db        Database
	jwtSecret []byte
	pk        *rsa.PublicKey

	loginView          *View
	mailboxView        *View
	errorView          *View
	changePasswordView *View
}

type AuthenticatedHandler = func(w http.ResponseWriter, r *http.Request, u *Usr)

func (wa *wa) renderError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	wa.errorView.Render(w, err)
}

func (wa *wa) root(w http.ResponseWriter, r *http.Request, u *Usr) {
	http.Redirect(w, r, "/mailbox/INBOX", http.StatusFound)
}

func (wa *wa) delete(w http.ResponseWriter, r *http.Request, u *Usr) {
	email := r.FormValue("email")
	if email == u.Email {
		wa.renderError(w, errors.New("You cannot delete yourself"))
		return
	}
	err := wa.db.DeleteUser(email)
	if err != nil {
		wa.renderError(w, err)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (wa *wa) add(w http.ResponseWriter, r *http.Request, u *Usr) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	_, err := wa.lg.NewUser(email, password, false)
	if err != nil {
		wa.renderError(w, err)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

type UserClaims struct {
	jwt.StandardClaims
	*Usr
}

func (c UserClaims) Valid() error {
	return c.StandardClaims.Valid()
}

func (wa *wa) login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	if email == "" {
		wa.loginView.Render(w, nil)
		return
	}

	usr, err := wa.lg.Login(email, password)
	if err != nil {
		wa.loginView.Render(w, err.Error())
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		jwt.StandardClaims{},
		usr,
	})
	tokenString, err := token.SignedString(wa.jwtSecret)
	if err != nil {
		wa.loginView.Render(w, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     GetString(JwtCookieNameKey),
		Value:    tokenString,
		HttpOnly: true,
		Secure:   GetBool(WebAdminUseTlsKey),
		Domain:   GetString(ServerNameKey),
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (wa *wa) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     GetString(JwtCookieNameKey),
		Value:    "bogus",
		Expires:  time.Now(),
		HttpOnly: true,
		Secure:   GetBool(WebAdminUseTlsKey),
		Domain:   GetString(ServerNameKey),
	})
	wa.loginView.Render(w, nil)
	w.WriteHeader(200)
}

func (wa *wa) checkAdmin(next AuthenticatedHandler) http.Handler {
	return wa.checkLogin(func(w http.ResponseWriter, r *http.Request, user *Usr) {
		if !user.Admin {
			wa.renderError(w, errors.New("You are not an administrator"))
			return
		}
		next(w, r, user)
	})
}

func (wa *wa) checkLogin(next AuthenticatedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, e := r.Cookie(GetString(JwtCookieNameKey))
		if e == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		var claims UserClaims
		_, err := jwt.ParseWithClaims(cookie.Value, &claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return wa.jwtSecret, nil
		})
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		err = claims.Valid()
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		next(w, r, claims.Usr)
	})
}

func (wa *wa) rotateJwt(w http.ResponseWriter, r *http.Request, u *Usr) {
	wa.jwtSecret = generateAndSaveJwtSecret()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (wa *wa) changePassword(w http.ResponseWriter, r *http.Request, u *Usr) {
	data := struct {
		Message string
	}{}
	if r.Method == http.MethodPost {
		password := r.FormValue("password")
		password2 := r.FormValue("password2")
		err := wa.lg.ChangePassword(u.Email, password, password2)
		if err != nil {
			data.Message = err.Error()
		} else {
			data.Message = "Password successfully changed"
		}
	}
	wa.changePasswordView.Render(w, data)
}

func (wa *wa) layoutData(u *Usr) (*LayoutData, error) {
	mbxes, e := wa.db.GetMailboxes(true, u.Id)
	if e != nil {
		return nil, e
	}
	return &LayoutData{
		CurrentUser: u,
		Mailboxes:   mbxes,
	}, nil
}

func (wa *wa) mailbox(w http.ResponseWriter, r *http.Request, u *Usr) {
	mbxName := mux.Vars(r)["name"]
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	mbx, e := wa.db.GetMailboxByName(mbxName, u.Id)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	msgs, e := wa.db.GetMessages(mbx.Id, -1, -1)
	data := struct {
		LayoutData
		Messages []*Msg
	}{
		*ld,
		msgs,
	}
	wa.mailboxView.Render(w, data)
}

func StartWebAdmin(lg Login, db Database, config *tls.Config, pk *rsa.PublicKey) {
	// Generate or read secret for JWT auth
	jwtSecret, e := ioutil.ReadFile(GetString(JwtTokenSecretFileKey))
	if os.IsNotExist(e) {
		jwtSecret = generateAndSaveJwtSecret()
	} else if e != nil {
		log.Fatal(e)
	}

	// Read the templates
	tp := template.New("html")
	e = filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
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
		loginView:          NewView("login.html", "templates/login.html"),
		mailboxView:        NewView("index.html", "templates/mailbox.html"),
		changePasswordView: NewView("index.html", "templates/change_password.html"),
	}

	if e != nil {
		log.Fatal(e)
	}
	router := mux.NewRouter()
	router.HandleFunc("/login", webAdmin.login)
	router.HandleFunc("/logout", webAdmin.logout)
	router.Handle("/changePassword", webAdmin.checkLogin(webAdmin.changePassword))
	router.Handle("/mailbox/{name}", webAdmin.checkLogin(webAdmin.mailbox))
	router.Handle("/", webAdmin.checkLogin(webAdmin.root))

	admin := router.PathPrefix("/admin/").Subrouter()
	admin.Handle("/addUser", webAdmin.checkAdmin(webAdmin.add))
	admin.Handle("/deleteUser", webAdmin.checkAdmin(webAdmin.delete))
	admin.Handle("/rotateJwt", webAdmin.checkAdmin(webAdmin.rotateJwt))

	server := &http.Server{Addr: GetString(WebAdminAddressKey), Handler: router}

	go func() {
		l, e := net.Listen("tcp", server.Addr)
		if e != nil {
			log.Fatal(e)
		}
		if GetBool(WebAdminUseTlsKey) {
			l = tls.NewListener(l, config)
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
	e = ioutil.WriteFile(GetString(JwtTokenSecretFileKey), jwtSecret, 0700)
	if e != nil {
		log.Fatal(e)
	}
	return jwtSecret
}

/*
func (wa *wa) runHealthChecks() HealthChecksViewModel {
	txtRecords, _ := net.LookupTXT("mx._domainkey." + GetString(DomainKey))
	actual := ""
	if len(txtRecords) > 0 {
		actual = txtRecords[0]
	}
	pkb, _ := x509.MarshalPKIXPublicKey(wa.pk)
	buf := new(bytes.Buffer)
	_, _ = base64.NewEncoder(base64.StdEncoding, buf).Write(pkb)
	expected := "v=dkim1; k=rsa; p=" + buf.String()
	return HealthChecksViewModel{
		TxtRecordIs:       actual,
		TxtRecordShouldBe: expected,
	}
}
*/
