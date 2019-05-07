package web

import (
	"henrymail/database"
	"henrymail/models"
	"net/http"
)

func (wa *wa) security(w http.ResponseWriter, r *http.Request, u *models.User) {
	data, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	wa.securityView.render(w, struct {
		layoutData
	}{
		layoutData: *data,
	})
}

func (wa *wa) rotateJwt(w http.ResponseWriter, r *http.Request, u *models.User) {
	key, e := models.KeyByName(database.DB, JwtSecretKeyName)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	e = key.Delete(database.DB)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	http.Redirect(w, r, "security", http.StatusFound)
}
