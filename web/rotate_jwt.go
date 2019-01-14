package web

import (
	"henrymail/model"
	"net/http"
)

func (wa *wa) rotateJwt(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	wa.jwtSecret = generateAndSaveJwtSecret()
	http.Redirect(w, r, "/", http.StatusFound)
}
