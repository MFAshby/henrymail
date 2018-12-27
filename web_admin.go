package main

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"html/template"
	"log"
	"net/http"
	"time"
)

type wa struct {
	lg Login
	tp *template.Template
	db Database
}

// ViewModels, defining the data how it is displayed
type IndexViewModel struct {
	CurrentUser *Usr
	Users       []*Usr
}

type LoginViewModel struct {
	Error string
}

type ErrorViewModel struct {
	Error string
}

// TODO this should sbe somewhere better than in the binary
//  it should also sbe rotateable.
var someSecret = []byte("Some secret yo")

const authCookieName = "jwt_auth"

type AuthenticatedHandler = func(w http.ResponseWriter, r *http.Request, u *Usr)

func (wa *wa) renderLogin(w http.ResponseWriter, errorMessage string) {
	if err := wa.tp.ExecuteTemplate(w, "login.html", LoginViewModel{
		Error: errorMessage,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (wa *wa) renderRoot(w http.ResponseWriter, u *Usr, users []*Usr) {
	if err := wa.tp.ExecuteTemplate(w, "index.html", IndexViewModel{
		CurrentUser: u,
		Users:       users,
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
	_, err := wa.lg.NewUser(email, password)
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

	if _, err := wa.lg.Login(email, password); err != nil {
		wa.renderLogin(w, err.Error())
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
	})
	tokenString, err := token.SignedString(someSecret)
	if err != nil {
		wa.renderLogin(w, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    tokenString,
		HttpOnly: true,
		Secure:   true,
		Domain:   GetString(DomainKey),
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (wa *wa) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    authCookieName,
		Value:   "",
		Expires: time.Now(),
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func checkAuth(next AuthenticatedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, e := r.Cookie(authCookieName)
		if e == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return someSecret, nil
		})
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		err = claims.Valid()
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		email, ok := claims["email"].(string)
		if !ok {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		user := &Usr{Email: email}
		next(w, r, user)
	})
}

func StartWebAdmin(lg Login, db Database) {
	// TODO bundle these into the binary for release builds?
	tp := template.Must(template.New("html").
		ParseFiles("templates/index.html",
			"templates/login.html",
			"templates/error.html"))

	webAdmin := wa{
		lg: lg,
		tp: tp,
		db: db,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/login", webAdmin.login)
	mux.HandleFunc("/logout", webAdmin.logout)
	mux.Handle("/deleteUser", checkAuth(webAdmin.delete))
	mux.Handle("/addUser", checkAuth(webAdmin.add))
	mux.Handle("/", checkAuth(webAdmin.rootGet))

	go func() {
		addr := GetString(WebAdminAddressKey)
		log.Println("Started admin web server at ", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatal(err)
		}
	}()
}
