package web

import (
	"errors"
	"henrymail/model"
	"net/http"
)

func (wa *wa) delete(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	username := r.FormValue("username")
	if username == u.Username {
		wa.renderError(w, errors.New("You cannot delete yourself"))
		return
	}
	err := wa.db.DeleteUser(username)
	if err != nil {
		wa.renderError(w, err)
	} else {
		http.Redirect(w, r, "users", http.StatusFound)
	}
}

func (wa *wa) add(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	_, err := wa.lg.NewUser(username, password, false)
	if err != nil {
		wa.renderError(w, err)
	} else {
		http.Redirect(w, r, "users", http.StatusFound)
	}
}

func (wa *wa) users(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	usrs, e := wa.db.GetUsers()
	if e != nil {
		wa.renderError(w, e)
		return
	}
	data := struct {
		LayoutData
		Users []*model.Usr
	}{
		*ld,
		usrs,
	}
	wa.usersView.Render(w, data)
}
