package render

import (
	"net/http"
)

type Render interface {
	Render(w http.ResponseWriter, code int) error
	WriteContentType(w http.ResponseWriter)
}

func WriteContentType(w http.ResponseWriter, value string) error {
	w.Header().Set("Content-Type", value)
	return nil
}
