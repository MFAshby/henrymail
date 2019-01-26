package web

import (
	"henrymail/model"
	"net/http"
)

func (wa *wa) changePassword(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	layoutData, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	data := struct {
		LayoutData
		Message string
	}{
		*layoutData,
		"",
	}
	if r.Method == http.MethodPost {
		password := r.FormValue("password")
		password2 := r.FormValue("password2")
		err := wa.lg.ChangePassword(u.Username, password, password2)
		if err != nil {
			data.Message = err.Error()
		} else {
			data.Message = "Password successfully changed"
		}
	}
	wa.changePasswordView.Render(w, data)
}
