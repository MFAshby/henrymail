package main

import (
	ev "github.com/asaskevich/EventBus"
	"html/template"
	"log"
	"net/http"
)

type wa struct {
	bus ev.Bus
	lg  Login
	tp  *template.Template
}

func (webAdmin *wa) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if err := webAdmin.tp.ExecuteTemplate(w, "login.html", nil); err != nil {
			log.Println(err)
		}
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	// Check the auth, issue a
	println("email ", email, "password", password)
}

func (*wa) root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusUnauthorized)
}

func StartWebAdmin(bus ev.Bus, lg Login) {
	tp := template.Must(template.New("html").
		ParseFiles("templates/index.html", "templates/login.html"))
	webAdmin := wa{
		bus: bus,
		lg:  lg,
		tp:  tp,
	}
	http.HandleFunc("/login", webAdmin.login)
	http.HandleFunc("/", webAdmin.root)
	go func() {
		addr := GetString(WebAdminAddressKey)
		println("Started admin web server at ", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	}()
}
