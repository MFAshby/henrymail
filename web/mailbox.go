package web

import (
	"github.com/gorilla/mux"
	"henrymail/model"
	"net/http"
)

func (wa *wa) mailbox(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	mbx, msgs, e := wa.msgs(mux.Vars(r)["name"], u)
	data := struct {
		LayoutData
		Mailbox  *model.Mbx
		Messages []*model.Msg
	}{
		*ld,
		mbx,
		msgs,
	}
	wa.mailboxView.Render(w, data)
}
