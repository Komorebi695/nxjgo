package render

import (
	"net/http"
	"text/template"
)

type HTML struct {
	Template   *template.Template
	Name       string
	Data       any
	IsTemplate bool
}

func (h *HTML) Render(w http.ResponseWriter, code int) error {
	h.WriteContentType(w)
	w.WriteHeader(code)
	if !h.IsTemplate {
		_, err := w.Write([]byte(h.Data.(string)))
		return err
	}
	err := h.Template.ExecuteTemplate(w, h.Name, h.Data)
	return err
}

func (h *HTML) WriteContentType(w http.ResponseWriter) {
	_ = WriteContentType(w, "text/html; charset=utf-8")
}

type HTMLRender struct {
	Template *template.Template
}
