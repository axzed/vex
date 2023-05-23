package render

import (
	"github.com/axzed/vex/internal/bytesconv"
	"html/template"
	"net/http"
)

type HTML struct {
	Data       any
	Name       string
	Template   *template.Template
	IsTemplate bool
}

type HTMLRender struct {
	Template *template.Template
}

func (h *HTML) Render(w http.ResponseWriter, code int) error {
	// 写入响应头
	h.WriteContentType(w)
	w.WriteHeader(code)
	// 判断是否是模板
	if h.IsTemplate {
		// 执行模板
		err := h.Template.ExecuteTemplate(w, h.Name, h.Data)
		return err
	}
	// 写入响应体
	_, err := w.Write(bytesconv.StringToBytes(h.Data.(string)))
	return err
}

func (h *HTML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/html; charset=utf-8")
}
