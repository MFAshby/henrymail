package web

import (
	"github.com/gorilla/mux"
	"henrymail/model"
	"net/http"
	"strconv"
)

func (wa *wa) message(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	name := mux.Vars(r)["name"]

	id, e := strconv.Atoi(mux.Vars(r)["id"])
	if e != nil {
		wa.renderError(w, e)
		return
	}
	_, msgs, e := wa.msgs(name, u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	var sel *model.Msg
	for _, msg := range msgs {
		if msg.Id == int64(id) {
			sel = msg
		}
	}
	wa.messageView.Render(w, struct {
		LayoutData
		Message *model.Msg
	}{
		*ld,
		sel,
	})
}
