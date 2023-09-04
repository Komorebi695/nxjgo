package nxjgo

import (
	"errors"
	"github.com/Komorebi695/nxjgo/binding"
	nxjLog "github.com/Komorebi695/nxjgo/log"
	"github.com/Komorebi695/nxjgo/render"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"text/template"
)

const defaultMultipartMemory = 32 << 20 // 32 MB

type Context struct {
	W          http.ResponseWriter
	R          *http.Request
	engine     *Engine
	queryCache url.Values
	formCache  url.Values
	// There are attributes in the implementation parameters,
	// but the corresponding structure does not have them, and an error is reported.
	DisallowUnknownFields bool
	// There are some attributes in the implementation structure,
	// but they are not included in the parameters, and an error is reported.
	IsValidate bool
	StatusCode int
	Logger     *nxjLog.Logger
	Keys       map[string]any
	mu         sync.RWMutex
	sameSite   http.SameSite
}

func (c *Context) GetCookie(name string) (string, error) {
	cookie, err := c.R.Cookie(name)
	if err != nil {
		return "", err
	}
	if cookie != nil {
		return cookie.Value, nil
	}
	return "", nil
}

func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(c.W, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		SameSite: c.sameSite,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}
	c.Keys[key] = value
	c.mu.Unlock()
}

func (c *Context) Get(key string) (any, bool) {
	//c.mu.RLocker()
	//defer c.mu.RUnlock()
	v, ok := c.Keys[key]
	return v, ok
}

func (c *Context) SetBasicAuth(username, password string) {
	c.R.Header.Set("Authorization", "Basic "+BasicAuth(username, password))
}

func (c *Context) FormFile(name string) *multipart.FileHeader {
	file, header, err := c.R.FormFile(name)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	return header
}

func (c *Context) FormFiles(name string) []*multipart.FileHeader {
	multipartForm, err := c.MultipartForm()
	if err != nil {
		log.Println(err)
		return make([]*multipart.FileHeader, 0)
	}
	return multipartForm.File[name]
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.R.ParseMultipartForm(defaultMultipartMemory)
	return c.R.MultipartForm, err
}

func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func (c *Context) PostFormMap(key string) map[string]string {
	dicts, _ := c.GetPostFormMap(key)
	return dicts
}

func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initPostFormCache()
	return c.get(c.formCache, key)
}

func (c *Context) GetPostForm(key string) string {
	c.initPostFormCache()
	return c.formCache.Get(key)
}

func (c *Context) GetPostFormArray(key string) ([]string, bool) {
	c.initPostFormCache()
	values, ok := c.formCache[key]
	return values, ok
}

func (c *Context) PostFormArray(key string) []string {
	values, _ := c.GetPostFormArray(key)
	return values
}

func (c *Context) initPostFormCache() {
	if c.R != nil {
		if err := c.R.ParseMultipartForm(defaultMultipartMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formCache = c.R.PostForm
	} else {
		c.formCache = make(url.Values)
	}
}

func (c *Context) QueryMap(key string) map[string]string {
	dicts, _ := c.GetQueryMap(key)
	return dicts
}

func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.queryCache, key)
}

func (c *Context) get(cache map[string][]string, key string) (map[string]string, bool) {
	// user[id]=1&user[name]=ly
	dicts := make(map[string]string)
	exist := false
	for k, v := range cache {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = v[0]
			}
		}
	}
	return dicts, exist
}

func (c *Context) GetDefaultQuery(key string, defaultValue string) string {
	values, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return values[0]
}

func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.initQueryCache()
	values, ok := c.queryCache[key]
	return values, ok
}

func (c *Context) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

func (c *Context) Render(code int, r render.Render) error {
	err := r.Render(c.W, code)
	// 多次调用产生警告: superfluous response.WriteHeader call
	c.StatusCode = code
	return err
}

func (c *Context) Redirect(status int, location string) error {
	err := c.Render(status, &render.Redirect{
		Code:     status,
		Request:  c.R,
		Location: location,
	})
	return err
}

func (c *Context) File(filePath string) {
	http.ServeFile(c.W, c.R, filePath)
}

func (c *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}

func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)
	c.R.URL.Path = filepath
	http.FileServer(fs).ServeHTTP(c.W, c.R)
}

//	 String old way
//		func (c *Context) String(status int, format string, values ...any) (err error) {
//			plainContentType := "text/plain; charset=utf-8"
//			c.W.Header().Set("Content-Type", plainContentType)
//			c.W.WriteHeader(status)
//			if len(values) > 0 {
//				_, err = fmt.Fprintf(c.W, format, values...)
//				return err
//			}
//			_, err = c.W.Write(StringToBytes(format))
//			return err
//		}
//
// String .
func (c *Context) String(status int, format string, values ...any) (err error) {
	err = c.Render(status, &render.String{
		Format: format,
		Data:   values,
	})
	return err
}

// HTML old way
//
//	func (c *Context) HTML(status int, html string) error {
//		c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
//		c.W.WriteHeader(status)
//		_, err := c.W.Write([]byte(html))
//		return err
//	}
//
// HTML .
func (c *Context) HTML(status int, html string) error {
	err := c.Render(status, &render.HTML{Data: html, IsTemplate: false})
	return err
}

// Template old way
//
//	func (c *Context) Template(name string, data any) error {
//		c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
//		err := c.engine.HTMLRender.Template.ExecuteTemplate(c.W, name, data)
//		return err
//	}
//
// Template .
func (c *Context) Template(name string, data any) error {
	err := c.Render(http.StatusOK, &render.HTML{
		Name: name, Data: data,
		Template:   c.engine.HTMLRender.Template,
		IsTemplate: true,
	})
	return err
}

func (c *Context) HTMLTemplate(name string, data any, filenames ...string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	_, err := t.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	_, err := t.ParseGlob(pattern)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

// JSON old way
//
//	func (c *Context) JSON(status int, data any) error {
//		c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
//		c.W.WriteHeader(status)
//		jsonData, err := &json.Marshal(data)
//		if err != nil {
//			return err
//		}
//		_, err = c.W.Write(jsonData)
//		return err
//	}
//
// JSON .
func (c *Context) JSON(status int, data any) error {
	err := c.Render(status, &render.JSON{Data: data})
	return err
}

// XML old way
//
//	func (c *Context) XML(status int, data any) error {
//		c.W.Header().Set("Content-Type", "application/xml; charset=utf-8")
//		c.W.WriteHeader(status)
//		//xmlData, err := &xml.Marshal(data)
//		//if err != nil {
//		//	return err
//		//}
//		//_, err = c.W.Write(xmlData)
//		err := xml.NewEncoder(c.W).Encode(data)
//		return err
//	}
//
// XML .
func (c *Context) XML(status int, data any) error {
	err := c.Render(status, &render.XML{Data: data})
	return err
}

func (c *Context) MustBindWith(obj any, bind binding.Binding) error {
	if err := c.ShowBind(obj, bind); err != nil {
		c.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

func (c *Context) ShowBind(obj any, bind binding.Binding) error {
	return bind.Bind(c.R, obj)
}

func (c *Context) BindJson(obj any) error {
	json := binding.JSON
	json.DisallowUnknownFields = c.DisallowUnknownFields
	json.IsValidate = c.IsValidate
	return c.MustBindWith(obj, json)
	//body := c.R.Body
	//if c.R == nil || body == nil {
	//	return errors.New("invalid request")
	//}
	//decoder := json.NewDecoder(body)
	//if c.DisallowUnknownFields {
	//	decoder.DisallowUnknownFields()
	//}
	//if c.IsValidate {
	//	if err := c.validateRequireParam(obj, decoder); err != nil {
	//		return err
	//	}
	//} else {
	//	if err := decoder.Decode(obj); err != nil {
	//		return err
	//	}
	//}
	//return validate(obj)
}

func (c *Context) BindXML(obj any) error {
	return c.MustBindWith(obj, binding.XML)
}

func (c *Context) Fail(code int, msg string) {
	_ = c.String(code, msg)
}

func (c *Context) ErrorHandle(err error) {
	code, data := c.engine.errorHandler(err)
	_ = c.JSON(code, data)
}
