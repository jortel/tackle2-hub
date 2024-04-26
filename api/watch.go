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
	subject string
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
	subject := ctx.Param(Wildcard)
	subject = strings.Split(subject, "/")[0]
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
			subject: subject,
			methods: methods,
		})
}

// Publish event.
func (h *WatchHandler) Publish(ctx *gin.Context) {
	if len(ctx.Errors) > 0 {
		return
	}
	id, _ := strconv.Atoi(ctx.Param(ID))
	if id == 0 {
		return
	}
	kind := strings.Split(
		ctx.Request.URL.Path,
		"/")[0]
	for _, p := range h.Watches {
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

// Event watch event.
type Event struct {
	Method string `json:"method"`
	Kind   string
	ID     uint `json:"ID"`
}
