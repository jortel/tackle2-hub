package api

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	qf "github.com/konveyor/tackle2-hub/api/filter"
	"gorm.io/gorm"
)

// Routes
const (
	WatchRoot = "/watch"
)

// Event watch event.
type Event struct {
	Method string `json:"method"`
	Object any    `json:"object"`
}

// Watch event pusher.
type Watch struct {
	id         int
	socket     *websocket.Conn
	writer     io.Writer
	queue      chan *Event
	done       chan int
	primer     Primer
	collection string
	methods    []string
}

// match selectors.
func (w *Watch) match(collection, method string) (matched bool) {
	if w.collection != collection {
		return
	}
	if len(w.methods) == 0 {
		matched = true
		return
	}
	for _, m := range w.methods {
		if m == method {
			matched = true
			break
		}
	}
	return
}

// forward events.
func (w *Watch) send(event *Event) {
	defer func() {
		_ = recover()
	}()
	w.queue <- event
	return
}

//  begin forwarding events.
func (w *Watch) begin() {
	w.done = make(chan int, 100)
	go func() {
		drain := false
		var err error
		defer func() {
			Log.Info("Watch ended.", "id", w.id)
			w.done <- 0
		}()
		for {
			select {
			case event := <-w.queue:
				if event == nil {
					w.done <- 0
					return
				}
				if drain {
					continue
				}
				if err != nil {
					drain = true
					w.done <- 0
					continue
				}
				var writer io.Writer
				if w.socket != nil {
					writer, err = w.socket.NextWriter(websocket.TextMessage)
					if err != nil {
						continue
					}
				} else {
					writer = w.writer
				}
				je := json.NewEncoder(writer)
				err = je.Encode(event)
				if err != nil {
					continue
				}
				closer, cast := writer.(io.WriteCloser)
				if cast {
					_ = closer.Close()
					continue
				}
				flusher, cast := writer.(http.Flusher)
				if cast {
					flusher.Flush()
					continue
				}
			}
		}
	}()
}

type Primer = gin.HandlerFunc

// WatchHandler handler.
type WatchHandler struct {
	BaseHandler
	Watches []*Watch
	mutex   sync.Mutex
	nextId  int
}

// Add a watch.
func (h *WatchHandler) Add(ctx *gin.Context, primer Primer) {
	collection, err := h.collection(ctx, "")
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	methods, afterId, err := h.filter(ctx)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	w := &Watch{
		id:         h.nextId,
		queue:      make(chan *Event, 10),
		writer:     ctx.Writer,
		collection: collection,
		primer:     primer,
	}
	err = h.upgrade(ctx, w)
	if err != nil {
		return
	}
	hdr := ctx.Writer.Header()
	hdr.Set("Connection", "Keep-Alive")
	ctx.Status(http.StatusOK)
	w.begin()
	rtx := WithContext(ctx)
	err = h.snapshot(rtx.DB, afterId, w)
	if err != nil {
		_ = ctx.Error(err)
		h.end(w)
		return
	}
	w.methods = methods
	h.mutex.Lock()
	h.nextId++
	h.Watches = append(h.Watches, w)
	h.mutex.Unlock()
	Log.Info("Watch created.", "id", w.id)
	_ = <-w.done
	close(w.queue)
	Log.Info("Watch queue closed.", "id", w.id)
	h.mutex.Lock()
	h.delete(w)
	h.mutex.Unlock()
	Log.Info("Watch deleted.", "id", w.id)
}

func (h *WatchHandler) kind(object any) (kind string) {
	t := reflect.TypeOf(object)
	kind = t.Name()
	return
}

// Publish event.
func (h *WatchHandler) Publish(ctx *gin.Context) {
	if len(ctx.Errors) > 0 {
		return
	}
	if len(h.Watches) == 0 {
		return
	}
	rtx := WithContext(ctx)
	object := rtx.Response.Body
	method := ctx.Request.Method
	collection, err := h.collection(ctx, method)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	h.mutex.Lock()
	watches := make([]*Watch, len(h.Watches))
	copy(watches, h.Watches)
	h.mutex.Unlock()
	for _, w := range watches {
		if !w.match(collection, method) {
			continue
		}
		w.send(
			&Event{
				Method: method,
				Object: object,
			})
	}
}

// Shutdown ends all watches.
func (h *WatchHandler) Shutdown() {
	h.mutex.Lock()
	watches := make([]*Watch, len(h.Watches))
	copy(watches, h.Watches)
	h.mutex.Unlock()
	for _, w := range watches {
		h.end(w)
	}
}

// end the session.
func (h *WatchHandler) end(w *Watch) {
	Log.Info("Watch end requested.", "id", w.id)
	defer func() {
		_ = recover()
	}()
	w.done <- 0
	return
}

// delete watch.
func (h *WatchHandler) delete(unwanted *Watch) {
	var kept []*Watch
	for _, w := range h.Watches {
		if w.id != unwanted.id {
			kept = append(kept, w)
		}
	}
	h.Watches = kept
}

// upgrade the connection when requested.
func (h *WatchHandler) upgrade(ctx *gin.Context, watch *Watch) (err error) {
	hdr := ctx.Request.Header.Get(Connection)
	hdr = strings.ToUpper(hdr)
	upgrade := hdr == "UPGRADE"
	if !upgrade {
		return
	}
	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(
		ctx.Writer,
		ctx.Request,
		nil)
	if err == nil {
		watch.socket = socket
	}
	return
}

// filter returns a list of
func (h *WatchHandler) filter(ctx *gin.Context) (methods []string, afterId uint, err error) {
	filter, err := qf.New(ctx,
		[]qf.Assert{
			{Field: "id", Kind: qf.LITERAL},
			{Field: "method", Kind: qf.STRING},
		})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	method, found := filter.Field("method")
	if found {
		for _, m := range method.Value {
			methods = append(
				methods,
				m.Value)
		}
	}
	id, found := filter.Field("id")
	if found {
		switch id.Operator.Value {
		case string(qf.EQ),
			string(qf.GT):
			v := id.Value[0].Value
			u, nErr := strconv.ParseUint(v, 10, 64)
			if nErr == nil {
				err = nErr
				return
			}
			afterId = uint(u)
		}
	}
	return
}

// snapshot sends inital set of events.
func (h *WatchHandler) snapshot(db *gorm.DB, afterId uint, w *Watch) (err error) {
	if !w.match(w.collection, http.MethodPost) {
		return
	}
	ctx := &gin.Context{}
	ctx.Writer = &ResponseWriter{}
	rtx := WithContext(ctx)
	rtx.DB = db
	w.primer(ctx)
	err = ctx.Err()
	if err != nil {
		return
	}
	body := rtx.Response.Body
	if body == nil {
		return
	}
	bt := reflect.TypeOf(body)
	switch bt.Kind() {
	case reflect.Slice:
		bv := reflect.ValueOf(body)
		for i := 0; i < bv.Len(); i++ {
			r := bv.Index(i).Interface()
			if r, cast := r.(interface{ Id() uint }); cast {
				id := r.Id()
				if id > afterId {
					continue
				}
			}
			w.send(
				&Event{
					Method: http.MethodPost,
					Object: r,
				})
		}
	default:
	}
	return
}

// collection returns the collection part of the path.
func (h *WatchHandler) collection(ctx *gin.Context, method string) (kind string, err error) {
	path := ctx.Request.URL.Path
	path = strings.TrimPrefix(path, "/")
	part := strings.Split(
		path,
		"/")
	p := 0
	switch method {
	case http.MethodPost:
		p = 0
	case http.MethodPut:
		p = 1
	case http.MethodPatch:
		p = 1
	}
	if len(part) < p {
		_ = ctx.Error(&BadRequestError{})
		return
	}
	slices.Reverse(part)
	kind = part[p]
	kind = strings.ToLower(kind)
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
