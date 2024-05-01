package api

import (
	"bufio"
	"net"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/konveyor/tackle2-hub/auth"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Context custom settings.
type Context struct {
	*gin.Context
	// DB client.
	DB *gorm.DB
	// User
	User string
	// Scopes
	Scopes []auth.Scope
	// Watch handler.
	Watch *WatchHandler
	// k8s Client
	Client client.Client
	// Response
	Response Response
}

// Response values.
type Response struct {
	Status int
	Body   interface{}
}

// Status sets the values to respond to the request with.
func (r *Context) Status(status int) {
	r.Response = Response{
		Status: status,
		Body:   nil,
	}
}

// Respond sets the values to respond to the request with.
func (r *Context) Respond(status int, body interface{}) {
	r.Response = Response{
		Status: status,
		Body:   body,
	}
}

// Fake returns a fake context.
func (r *Context) Fake() (fake *gin.Context) {
	fake = &gin.Context{}
	fake.Request = &http.Request{}
	fake.Request.URL = &url.URL{}
	fake.Writer = &ResponseWriter{}
	rtx := WithContext(fake)
	rtx.DB = r.DB
	rtx.Client = r.Client
	return
}

type ResponseWriter struct {
}

func (w *ResponseWriter) Header() (h http.Header) {
	h = make(http.Header)
	return
}

func (w *ResponseWriter) Unwrap() (r http.ResponseWriter) {
	return
}

func (w *ResponseWriter) reset(writer http.ResponseWriter) {
}

func (w *ResponseWriter) WriteHeader(code int) {
}

func (w *ResponseWriter) WriteHeaderNow() {
}

func (w *ResponseWriter) Write(data []byte) (n int, err error) {
	return
}

func (w *ResponseWriter) WriteString(s string) (n int, err error) {
	return
}

func (w *ResponseWriter) Status() (n int) {
	return
}

func (w *ResponseWriter) Size() (n int) {
	return
}

func (w *ResponseWriter) Written() (b bool) {
	return
}

func (w *ResponseWriter) Hijack() (conn net.Conn, r *bufio.ReadWriter, err error) {
	return
}

func (w *ResponseWriter) CloseNotify() (ch <-chan bool) {
	return
}

func (w *ResponseWriter) Flush() {
}

func (w *ResponseWriter) Pusher() (pusher http.Pusher) {
	return
}

// WithContext is a rich context.
func WithContext(ctx *gin.Context) (n *Context) {
	key := "RichContext"
	object, found := ctx.Get(key)
	if !found {
		n = &Context{}
		ctx.Set(key, n)
	} else {
		n = object.(*Context)
	}
	n.Context = ctx
	return
}

// Transaction handler.
func Transaction(ctx *gin.Context) {
	switch ctx.Request.Method {
	case http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete:
		rtx := WithContext(ctx)
		err := rtx.DB.Transaction(func(tx *gorm.DB) (err error) {
			db := rtx.DB
			rtx.DB = tx
			ctx.Next()
			rtx.DB = db
			if len(ctx.Errors) > 0 {
				err = ctx.Errors[0]
				ctx.Errors = nil
			}
			return
		})
		if err != nil {
			_ = ctx.Error(err)
		}
	}
}

// Render renders the response based on the Accept: header.
// Opinionated towards json.
func Render() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		rtx := WithContext(ctx)
		if rtx.Response.Body != nil {
			ctx.Negotiate(
				rtx.Response.Status,
				gin.Negotiate{
					Offered: BindMIMEs,
					Data:    rtx.Response.Body})
			return
		}
		ctx.Status(rtx.Response.Status)
	}
}
