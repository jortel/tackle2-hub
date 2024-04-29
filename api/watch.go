package api

import (
	"encoding/json"
	"io"
	"net/http"
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

// Watched resource.
type Watched interface {
	Id() (id uint)
}

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
	kind       string
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
				writer := w.writer
				if w.socket != nil {
					writer, err = w.socket.NextWriter(websocket.TextMessage)
					if err != nil {
						continue
					}
				}
				je := json.NewEncoder(writer)
				err = je.Encode(event)
				if err != nil {
					continue
				}
				flusher, cast := writer.(http.Flusher)
				if cast {
					flusher.Flush()
				}
			}
		}
	}()
}

// WatchHandler handler.
type WatchHandler struct {
	BaseHandler
	Watches []*Watch
	mutex   sync.Mutex
	nextId  int
}

// Add a watch.
func (h *WatchHandler) Add(ctx *gin.Context, kind string) {
	collection, err := h.collection(ctx, 1)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	methods, db, err := h.filter(ctx)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	w := &Watch{
		id:         h.nextId,
		queue:      make(chan *Event),
		writer:     ctx.Writer,
		collection: collection,
		kind:       kind,
	}
	err = h.upgrade(ctx, w)
	if err != nil {
		return
	}
	hdr := ctx.Writer.Header()
	hdr.Set("Connection", "Keep-Alive")
	ctx.Status(http.StatusOK)
	w.begin()
	err = h.snapshot(db, w)
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
	collection, err := h.collection(ctx, 0)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	method := ctx.Request.Method
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
func (h *WatchHandler) delete(w *Watch) {
	var kept []*Watch
	for i := range h.Watches {
		if w != h.Watches[i] {
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
func (h *WatchHandler) filter(ctx *gin.Context) (methods []string, db *gorm.DB, err error) {
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
	filter = filter.With("id")
	db = filter.Where(h.DB(ctx))
	return
}

// snapshot sends inital set of events.
func (h *WatchHandler) snapshot(db *gorm.DB, w *Watch) (err error) {
	if !w.match(w.collection, http.MethodPost) {
		return
	}
	var list []map[string]any
	db = db.Table(w.kind)
	err = db.Find(&list).Error
	if err != nil {
		return
	}
	for _, r := range list {
		w.send(
			&Event{
				Method: http.MethodPost,
				Object: r,
			})
	}
	return
}

// collection returns the collection part of the path.
func (h *WatchHandler) collection(ctx *gin.Context, p int) (kind string, err error) {
	path := ctx.Request.URL.Path
	path = strings.TrimPrefix(path, "/")
	part := strings.Split(
		path,
		"/")
	if len(part) < p {
		_ = ctx.Error(&BadRequestError{})
		return
	}
	kind = part[p]
	kind = strings.ToLower(kind)
	return
}
