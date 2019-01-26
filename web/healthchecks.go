package web

import (
	"henrymail/config"
	"henrymail/dkim"
	"henrymail/model"
	"net"
	"net/http"
)

func (wa *wa) healthChecks(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	txtRecords, _ := net.LookupTXT("mx._domainkey." + config.GetString(config.Domain))
	actual := ""
	if len(txtRecords) > 0 {
		actual = txtRecords[0]
	}

	dkimRecordString, e := dkim.GetDkimRecordString()
	if e != nil {
		wa.renderError(w, e)
		return
	}
	data := struct {
		LayoutData
		TxtRecordIs       string
		TxtRecordShouldBe string
	}{
		LayoutData:        *ld,
		TxtRecordIs:       actual,
		TxtRecordShouldBe: dkimRecordString,
	}
	wa.healthChecksView.Render(w, data)
}
