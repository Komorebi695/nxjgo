package nxjgo

import (
	"fmt"
	"github.com/Komorebi695/nxjgo/config"
	nxjLog "github.com/Komorebi695/nxjgo/log"
	"github.com/Komorebi695/nxjgo/render"
	"log"
	"net/http"
	"sync"
	"text/template"
)

const (
	ANY = "ANY"
)

type HandlerFunc func(ctx *Context)

type routerGroup struct {
	name               string
	handleFuncMap      map[string]map[string]HandlerFunc      // map[路由]map[方法]HandlerFunc
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc // map[路由]map[方法]MiddlewareFunc
	handleMethodMap    map[string][]string
	treeNode           *treeNode
	middlewares        []MiddlewareFunc
}

type MiddlewareFunc func(handlerFunc HandlerFunc) HandlerFunc

type router struct {
	groups []*routerGroup
	engine *Engine
}

func (r *router) Group(name string) *routerGroup {
	g := &routerGroup{
		name:               name,
		handleFuncMap:      make(map[string]map[string]HandlerFunc),
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
		handleMethodMap:    make(map[string][]string),
		treeNode:           &treeNode{name: "/", children: make([]*treeNode, 0)},
	}
	g.Use(r.engine.middle...)
	r.groups = append(r.groups, g)
	return g
}

func (r *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
}

//func (r *routerGroup) PostHandle(middlewareFunc ...MiddlewareFunc) {
//	r.postMiddlewares = append(r.postMiddlewares, middlewareFunc...)
//}

func (r *routerGroup) methodHandle(name string, method string, h HandlerFunc, ctx *Context) {
	// 组通用中间件
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			h = middlewareFunc(h)
		}
	}
	// 组路由级别
	funcMiddleware := r.middlewaresFuncMap[name][method]
	if funcMiddleware != nil {
		for _, middlewareFunc := range funcMiddleware {
			h = middlewareFunc(h)
		}
	}
	h(ctx)
}

func (r *routerGroup) handle(name string, method string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	_, ok := r.handleFuncMap[name]
	if !ok {
		r.handleFuncMap[name] = make(map[string]HandlerFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handleFuncMap[name][method]
	if ok {
		panic("There are duplicate routes.")
	}
	r.handleFuncMap[name][method] = handlerFunc
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...)
	r.handleMethodMap[method] = append(r.handleMethodMap[method], name)
	r.treeNode.Put(name)
}

func (r *routerGroup) Any(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, ANY, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Get(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Post(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Delete(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Put(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Patch(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Options(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handlerFunc, middlewareFunc...)
}

func (r *routerGroup) Head(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodHead, handlerFunc, middlewareFunc...)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.R = r
	ctx.W = w
	ctx.Logger = e.Logger
	e.httpRequestHandle(ctx)
	e.pool.Put(ctx)
}

type Engine struct {
	*router
	funcMap      template.FuncMap
	HTMLRender   render.HTMLRender
	pool         sync.Pool
	Logger       *nxjLog.Logger
	middle       []MiddlewareFunc
	errorHandler ErrorHandler
}

func New() *Engine {
	engine := &Engine{
		router: &router{},
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}

func Default() *Engine {
	engine := New()
	engine.funcMap = nil
	engine.HTMLRender = render.HTMLRender{}
	engine.Logger = nxjLog.Default()
	logPath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logPath.(string))
	}
	// 默认日志目录
	engine.Logger.SetLogPath("./log")
	engine.Use(Logging, Recovery)
	engine.router.engine = engine
	return engine
}

func (e *Engine) allocateContext() any {
	return &Context{engine: e}
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHtmlTemplate(t)
}

// LoadTemplateByConf 通过配置文件加载模板
func (e *Engine) LoadTemplateByConf() {
	pattern, ok := config.Conf.Template["pattern"]
	if !ok {
		panic("config pattern not exist")
	}
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern.(string)))
	e.SetHtmlTemplate(t)
}

func (e *Engine) SetHtmlTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: t}
}

func (e *Engine) Run(addr string) {
	http.Handle("/", e)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Engine) Handler() http.Handler {
	return e
}

func (e *Engine) httpRequestHandle(ctx *Context) {
	method := ctx.R.Method
	for _, g := range e.groups {
		routerName := SubStringLast(ctx.R.URL.Path, "/"+g.name)
		node := g.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			// 路由匹配上了
			handle, ok := g.handleFuncMap[node.routerName][ANY]
			if ok {
				g.methodHandle(node.routerName, ANY, handle, ctx)
				return
			}
			handle, ok = g.handleFuncMap[node.routerName][method]
			if ok {
				g.methodHandle(node.routerName, method, handle, ctx)
				return
			}
			ctx.W.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(ctx.W, "%s %s not allowed.", ctx.R.RequestURI, method)
			return
		}
	}
	ctx.W.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(ctx.W, "%s not found.", ctx.R.RequestURI)
}

func (e *Engine) Use(middle ...MiddlewareFunc) {
	e.middle = append(e.middle, middle...)
}

type ErrorHandler func(err error) (int, any)

func (e *Engine) RegisterErrorHandler(err ErrorHandler) {
	e.errorHandler = err
}
