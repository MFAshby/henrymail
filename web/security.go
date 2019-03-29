package web

import (
	"henrymail/model"
	"net/http"
)

func (wa *wa) security(w http.ResponseWriter, r *http.Request, u *model.Usr) {
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

func (wa *wa) rotateJwt(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	wa.jwtSecret = generateAndSaveJwtSecret()
	http.Redirect(w, r, "security", http.StatusFound)
}
