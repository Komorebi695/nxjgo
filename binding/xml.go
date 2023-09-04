package binding

import (
	"encoding/xml"
	"io"
	"net/http"
)

type xmlBinding struct{}

func (b xmlBinding) Name() string {
	return "xml"
}

func (b xmlBinding) Bind(r *http.Request, obj interface{}) error {
	return decodeXML(r.Body, obj)
}

func decodeXML(body io.ReadCloser, obj interface{}) error {
	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}
