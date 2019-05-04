package web

import (
	"henrymail/logic"
	"henrymail/models"
	"net/http"
)

func (wa *wa) changePassword(w http.ResponseWriter, r *http.Request, u *models.User) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	data := struct {
		layoutData
		Message string
	}{
		*ld,
		"",
	}
	if r.Method == http.MethodPost {
		oldPassword := r.FormValue("oldpassword")
		newPassword := r.FormValue("newpassword")
		newPassword2 := r.FormValue("newpassword2")
		err := logic.ChangePassword(wa.db, u.Username, oldPassword, newPassword, newPassword2)
		if err != nil {
			data.Message = err.Error()
		} else {
			data.Message = "Password successfully changed"
		}
	}
	wa.changePasswordView.render(w, data)
}
