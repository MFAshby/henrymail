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

	spfExpected := spf.GetSpfRecordString()
	spfActual := fetchSpfActual()

	dkimExpected, e := dkim.GetDkimRecordString()
	if e != nil {
		wa.renderError(w, e)
		return
	}
	dkimActual := fetchDkimActual()

	data := struct {
		layoutData
		DkimRecordIs       string
		DkimRecordShouldBe string
		SpfRecordIs        string
		SpfRecordShouldBe  string
	}{
		layoutData:         *ld,
		DkimRecordIs:       dkimActual,
		DkimRecordShouldBe: dkimExpected,
		SpfRecordIs:        spfActual,
		SpfRecordShouldBe:  spfExpected,
	}
	wa.healthChecksView.render(w, data)
}

func fetchDkimActual() string {
	dkimActual := ""
	dkimRecords, e := net.LookupTXT("mx._domainkey." + config.GetString(config.Domain))
	if e != nil {
		dkimActual = e.Error()
	} else {
		// Longer TXT records may be split
		dkimActual = strings.Join(dkimRecords, "")
	}
	return dkimActual
}

func fetchSpfActual() string {
	spfActual := ""
	spfRecords, e := net.LookupTXT(config.GetString(config.Domain))
	if e != nil {
		spfActual = e.Error()
	} else {
		for _, s := range spfRecords {
			if strings.Contains(s, "v=spf1") {
				spfActual = s
			}
		}
	}
	return spfActual
}
