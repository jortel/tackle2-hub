package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

// Routes
const (
	WatchRoot = "/watch"
)

// Watch event pusher.
type Watch struct {
	conn    *websocket.Conn
	kind    string
	methods []string
}

// send an event.
func (p *Watch) send(e *Event) (err error) {
	err = p.conn.WriteJSON(e)
	return
}

// end the session.
func (p *Watch) end() {
	_ = p.conn.Close()
}

// WatchHandler handler.
type WatchHandler struct {
	Watches []*Watch
}

// Shutdown ends all watches.
func (h *WatchHandler) Shutdown() {
	for _, p := range h.Watches {
		p.end()
	}
}

// Add a watch.
func (h *WatchHandler) Add(ctx *gin.Context) {
	filter := ctx.Param(Filter)
	methods := strings.Split(filter, ",")
	for i := range methods {
		methods[i] = strings.TrimSpace(methods[i])
		methods[i] = strings.ToUpper(methods[i])
	}
	kind, err := h.kind(ctx, 1)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(
		ctx.Writer,
		ctx.Request,
		nil)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	h.Watches = append(
		h.Watches,
		&Watch{
			conn:    conn,
			kind:    kind,
			methods: methods,
		})
}

// Publish event.
func (h *WatchHandler) Publish(ctx *gin.Context) {
	if len(ctx.Errors) > 0 {
		return
	}
	if len(h.Watches) == 0 {
		return
	}
	p := ctx.Param(ID)
	id, _ := strconv.Atoi(p)
	if id == 0 {
		return
	}
	kind, err := h.kind(ctx, 0)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	for _, p := range h.Watches {
		if p.kind != kind {
			continue
		}
		err := p.send(&Event{
			Method: ctx.Request.Method,
			Kind:   kind,
			ID:     uint(id),
		})
		if err != nil {
			h.End(p)
		}
	}
}

// End watch.
func (h *WatchHandler) End(p *Watch) {
	var kept []*Watch
	for i := range h.Watches {
		if p != h.Watches[i] {
			kept = append(kept, p)
		}
	}
	h.Watches = kept
}

func (h *WatchHandler) kind(ctx *gin.Context, p int) (kind string, err error) {
	part := strings.Split(
		ctx.Request.URL.Path,
		"/")
	if len(part) < p {
		_ = ctx.Error(&BadRequestError{})
		return
	}
	kind = part[p]
	kind = strings.ToLower(kind)
	return
}

// Event watch event.
type Event struct {
	Method string `json:"method"`
	Kind   string
	ID     uint `json:"ID"`
}
