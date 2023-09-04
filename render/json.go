package render

import (
	"encoding/json"
	"net/http"
)

type JSON struct {
	Data any
}

func (j *JSON) Render(w http.ResponseWriter, code int) error {
	j.WriteContentType(w)
	w.WriteHeader(code)
	jsonData, err := json.Marshal(j.Data)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonData)
	return err
}

func (j *JSON) WriteContentType(w http.ResponseWriter) {
	_ = WriteContentType(w, "application/json; charset=utf-8")
}
