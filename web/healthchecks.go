package web

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"henrymail/config"
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
	pkb, _ := x509.MarshalPKIXPublicKey(wa.pk)
	buf := new(bytes.Buffer)
	_, _ = base64.NewEncoder(base64.StdEncoding, buf).Write(pkb)
	expected := "v=dkim1; k=rsa; p=" + buf.String()

	data := struct {
		LayoutData
		TxtRecordIs       string
		TxtRecordShouldBe string
	}{
		LayoutData:        *ld,
		TxtRecordIs:       actual,
		TxtRecordShouldBe: expected,
	}
	wa.healthChecksView.Render(w, data)
}
