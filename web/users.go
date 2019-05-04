package web

import (
	"errors"
	"henrymail/logic"
	"henrymail/models"
	"net/http"
)

func (wa *wa) delete(w http.ResponseWriter, r *http.Request, u *models.User) {
	username := r.FormValue("username")
	if username == u.Username {
		wa.renderError(w, errors.New("You cannot delete yourself"))
		return
	}
	user, err := models.UserByUsername(wa.db, username)
	if err != nil {
		wa.renderError(w, err)
		return
	}
	err = user.Delete(wa.db)
	if err != nil {
		wa.renderError(w, err)
		return
	}

	http.Redirect(w, r, "users", http.StatusFound)
}

func (wa *wa) add(w http.ResponseWriter, r *http.Request, u *models.User) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	isadmin := r.FormValue("admin") == "admin"
	_, err := logic.NewUser(wa.db, username, password, isadmin)
	if err != nil {
		wa.renderError(w, err)
		return
	}

	http.Redirect(w, r, "users", http.StatusFound)
}

func (wa *wa) users(w http.ResponseWriter, r *http.Request, u *models.User) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	users, e := models.GetAllUser(wa.db)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	data := struct {
		layoutData
		Users []*models.User
	}{
		*ld,
		users,
	}
	wa.usersView.render(w, data)
}
