package binding

import (
	"encoding/xml"
	"io"
	"net/http"
)

type queryBinding struct{}

func (queryBinding) Name() string {
	return "query"
}

func (queryBinding) Bind(req *http.Request, obj any) error {
	return decodeQuery(req.Body, obj)
}

func decodeQuery(body io.ReadCloser, obj interface{}) error {
	decoder := xml.NewDecoder(body)

	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}
