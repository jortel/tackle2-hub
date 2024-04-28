package api

import (
	"net/http"
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

// Watched resource.
type Watched interface {
	Id() (id uint)
}

// Event watch event.
type Event struct {
	Method string `json:"method"`
	Kind   string `json:"kind"`
	ID     uint   `json:"ID"`
}

// Watch event pusher.
type Watch struct {
	id         int
	conn       *websocket.Conn
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

// send an event.
func (w *Watch) send(event *Event) (err error) {
	err = w.conn.WriteJSON(event)
	if err == nil {
		Log.Info(
			"Watch event sent.",
			"id",
			w.id,
			"event",
			event)
	}
	return
}

// end the session.
func (w *Watch) end() {
	_ = w.conn.Close()
	Log.Info("Watch ended.", "id", w.id)
}

// WatchHandler handler.
type WatchHandler struct {
	BaseHandler
	Watches []*Watch
	mutex   sync.Mutex
	nextId  int
}

// Shutdown ends all watches.
func (h *WatchHandler) Shutdown() {
	for _, w := range h.Watches {
		w.end()
	}
}

// Add a watch.
func (h *WatchHandler) Add(ctx *gin.Context, kind string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(
		ctx.Writer,
		ctx.Request,
		nil)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
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
		conn:       conn,
		collection: collection,
		kind:       kind,
	}
	err = h.snapshot(db, w)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	h.nextId++
	w.methods = methods
	h.Watches = append(h.Watches, w)
	Log.Info("Watch created.", "id", w.id)
}

// Publish event.
func (h *WatchHandler) Publish(ctx *gin.Context) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if len(ctx.Errors) > 0 {
		return
	}
	if len(h.Watches) == 0 {
		return
	}
	p := ctx.Param(ID)
	id, _ := strconv.ParseUint(p, 10, 64)
	if id == 0 {
		rtx := WithContext(ctx)
		r := rtx.Response.Body
		if r != nil {
			if watched, cast := r.(Watched); cast {
				n := watched.Id()
				id = uint64(n)
			}
		} else {
			return
		}
	}
	collection, err := h.collection(ctx, 0)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	method := ctx.Request.Method
	for _, w := range h.Watches {
		if !w.match(collection, method) {

		}
		err := w.send(
			&Event{
				Method: method,
				Kind:   w.kind,
				ID:     uint(id),
			})
		if err != nil {
			h.End(w)
		}
	}
}

// End watch.
func (h *WatchHandler) End(w *Watch) {
	var kept []*Watch
	for i := range h.Watches {
		if w != h.Watches[i] {
			kept = append(kept, w)
		} else {
			w.end()
		}
	}
	h.Watches = kept
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
	db = db.Select("ID")
	err = db.Find(&list).Error
	if err != nil {
		return
	}
	id := uint(0)
	for _, r := range list {
		idStr := r["ID"]
		switch n := idStr.(type) {
		case int:
			id = uint(n)
		case int32:
			id = uint(n)
		case int64:
			id = uint(n)
		}
		sErr := w.send(
			&Event{
				Method: http.MethodPost,
				Kind:   w.kind,
				ID:     id,
			})
		if sErr != nil {
			h.End(w)
		}
	}
	return
}

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
