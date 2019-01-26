package web

import (
	"henrymail/config"
	"henrymail/dkim"
	"henrymail/model"
	"henrymail/spf"
	"net"
	"net/http"
	"strings"
)

func (wa *wa) healthChecks(w http.ResponseWriter, r *http.Request, u *model.Usr) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}

	spfRecords, e := net.LookupTXT(config.GetString(config.Domain))
	if e != nil {
		wa.renderError(w, e)
		return
	}
	spfActual := ""
	for _, s := range spfRecords {
		if strings.Contains(s, "v=spf1") {
			spfActual = s
		}
	}
	spfExpected := spf.GetSpfRecordString()

	dkimRecords, e := net.LookupTXT("mx._domainkey." + config.GetString(config.Domain))
	if e != nil {
		wa.renderError(w, e)
		return
	}
	// Longer TXT records may be split
	dkimActual := strings.Join(dkimRecords, "")
	dkimExpected, e := dkim.GetDkimRecordString()
	if e != nil {
		wa.renderError(w, e)
		return
	}

	data := struct {
		LayoutData
		DkimRecordIs       string
		DkimRecordShouldBe string
		SpfRecordIs        string
		SpfRecordShouldBe  string
	}{
		LayoutData:         *ld,
		DkimRecordIs:       dkimActual,
		DkimRecordShouldBe: dkimExpected,
		SpfRecordIs:        spfActual,
		SpfRecordShouldBe:  spfExpected,
	}
	wa.healthChecksView.Render(w, data)
}
