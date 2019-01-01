package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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

type wa struct {
	tp        *template.Template
	lg        Login
	db        Database
	jwtSecret []byte
	pk        *rsa.PublicKey
}

// ViewModels, defining the data how it is displayed
type HealthChecksViewModel struct {
	TxtRecordShouldBe string
	TxtRecordIs       string
}

type IndexViewModel struct {
	CurrentUser  *Usr
	Users        []*Usr
	HealthChecks HealthChecksViewModel
}

type LoginViewModel struct {
	Error string
}

type ErrorViewModel struct {
	Error string
}

type AuthenticatedHandler = func(w http.ResponseWriter, r *http.Request, u *Usr)

const jwtCookieName = "jwt_auth"

func (wa *wa) renderLogin(w http.ResponseWriter, errorMessage string) {
	if err := wa.tp.ExecuteTemplate(w, "login.html", LoginViewModel{
		Error: errorMessage,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (wa *wa) renderRoot(w http.ResponseWriter, u *Usr, users []*Usr) {
	if err := wa.tp.ExecuteTemplate(w, "index.html", IndexViewModel{
		CurrentUser:  u,
		Users:        users,
		HealthChecks: wa.runHealthChecks(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (wa *wa) renderError(w http.ResponseWriter, err string) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := wa.tp.ExecuteTemplate(w, "error.html", ErrorViewModel{
		Error: err,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (wa *wa) rootGet(w http.ResponseWriter, r *http.Request, u *Usr) {
	users, e := wa.db.GetUsers()
	if e != nil {
		wa.renderError(w, e.Error())
		return
	}
	wa.renderRoot(w, u, users)
}

func (wa *wa) delete(w http.ResponseWriter, r *http.Request, u *Usr) {
	email := r.FormValue("email")
	if email == u.Email {
		wa.renderError(w, "You cannot delete yourself")
		return
	}
	err := wa.db.DeleteUser(email)
	if err != nil {
		wa.renderError(w, err.Error())
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (wa *wa) add(w http.ResponseWriter, r *http.Request, u *Usr) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	_, err := wa.lg.NewUser(email, password, false)
	if err != nil {
		wa.renderError(w, err.Error())
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (wa *wa) login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	if email == "" {
		wa.renderLogin(w, "")
		return
	}

	usr, err := wa.lg.Login(email, password)
	if err != nil {
		wa.renderLogin(w, err.Error())
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"admin": usr.Admin,
	})
	tokenString, err := token.SignedString(wa.jwtSecret)
	if err != nil {
		wa.renderLogin(w, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     jwtCookieName,
		Value:    tokenString,
		HttpOnly: true,
		//Secure:   true,
		//Domain:   GetString(DomainKey),
		Secure: false,
		Domain: "localhost",
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (wa *wa) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     jwtCookieName,
		Value:    "bogus",
		Expires:  time.Now(),
		HttpOnly: true,
		//Secure:   true,
		//Domain:   GetSool(WebAdminUseTlsKey),
		Secure: false,
		Domain: "localhost",
	})
	w.WriteHeader(200)
}

func (wa *wa) checkAdmin(next AuthenticatedHandler) http.Handler {
	return wa.checkLogin(func(w http.ResponseWriter, r *http.Request, user *Usr) {
		if !user.Admin {
			wa.renderError(w, "You are not an administrator")
			return
		}
		next(w, r, user)
	})
}

func (wa *wa) checkLogin(next AuthenticatedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, e := r.Cookie(jwtCookieName)
		if e == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return wa.jwtSecret, nil
		})
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		err = claims.Valid()
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		email, ok := claims["email"].(string)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		admin, ok := claims["admin"].(bool)
		if !ok || !admin {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		user := &Usr{
			Email: email,
			Admin: admin,
		}
		next(w, r, user)
	})
}

func (wa *wa) rotateJwt(w http.ResponseWriter, r *http.Request, u *Usr) {
	wa.jwtSecret = generateAndSaveJwtSecret()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (wa *wa) changePassword(w http.ResponseWriter, r *http.Request, u *Usr) {
	password := r.FormValue("password")
	password2 := r.FormValue("password2")
	err := wa.lg.ChangePassword(u.Email, password, password2)
	if err != nil {
		wa.renderError(w, err.Error())
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
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
		lg:        lg,
		tp:        tp,
		db:        db,
		jwtSecret: jwtSecret,
		pk:        pk,
	}
	//mux := http.NewServeMux()
	router := mux.NewRouter()
	router.HandleFunc("/login", webAdmin.login)
	router.HandleFunc("/logout", webAdmin.logout)
	router.Handle("/changePassword", webAdmin.checkLogin(webAdmin.changePassword))
	router.Handle("/", webAdmin.checkLogin(webAdmin.rootGet))

	admin := router.PathPrefix("/admin/").Subrouter()
	admin.Handle("/addUser", webAdmin.checkAdmin(webAdmin.add))
	admin.Handle("/deleteUser", webAdmin.checkAdmin(webAdmin.delete))
	admin.Handle("/rotateJwt", webAdmin.checkAdmin(webAdmin.rotateJwt))

	server := &http.Server{Addr: GetString(WebAdminAddressKey), Handler: router}

	go func() {
		if GetBool(WebAdminUseTlsKey) {
			l, e := net.Listen("tcp", server.Addr)
			if e != nil {
				log.Fatal(e)
			}
			tlsListener := tls.NewListener(l, config)
			log.Println("Started admin web server using TLS at ", server.Addr)
			if err := server.Serve(tlsListener); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Println("Started admin web server WITHOUT TLS at ", server.Addr)
			if err := server.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
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
