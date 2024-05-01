package api

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	liberr "github.com/jortel/go-utils/error"
	qf "github.com/konveyor/tackle2-hub/api/filter"
)

// Routes
const (
	WatchRoot = "/watch"
)

// Event watch event.
type Event struct {
	Method string `json:"method"`
	Object any    `json:"object"`
	reader io.Reader
}

// Watch event pusher.
type Watch struct {
	id         int
	socket     *websocket.Conn
	writer     io.Writer
	queue      chan io.Reader
	done       chan int
	collection string
	methods    []string
	primer     Primer
	builder    EventBuilder
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
func (w *Watch) send(reader io.Reader) {
	defer func() {
		r := recover()
		if err, cast := r.(error); cast {
			Log.Error(err, "Watch send failed.", "id", w.id)
		}
	}()
	w.queue <- reader
	return
}

// begin forwarding events.
func (w *Watch) run() {
	w.done = make(chan int, 100)
	go func() {
		drain := false
		var err error
		defer func() {
			Log.Info("Watch ended.", "id", w.id)
			w.done <- 0
		}()
		push := func(reader io.Reader) (end bool) {
			defer w.close(reader)
			if reader == nil {
				end = true
				return
			}
			if drain {
				_, _ = io.ReadAll(reader)
				return
			}
			if err != nil {
				drain = true
				return
			}
			var writer io.Writer
			if w.socket != nil {
				writer, err = w.socket.NextWriter(websocket.TextMessage)
				if err != nil {
					end = true
					return
				}
			} else {
				writer = w.writer
			}
			_, err = io.Copy(writer, reader)
			if err != nil {
				end = true
				return
			}
			closer, cast := writer.(io.WriteCloser)
			if cast {
				_ = closer.Close()
				return
			}
			flusher, cast := writer.(http.Flusher)
			if cast {
				flusher.Flush()
				return
			}
			return
		}
		for r := range w.queue {
			end := push(r)
			if end {
				w.done <- 0
				break
			}
		}
	}()
}

func (w *Watch) close(r io.Reader) {
	defer func() {
		_ = recover()
	}()
	if r, cast := r.(io.Closer); cast {
		_ = r.Close()
	}
}

// Primer fetches resources and stores in the
// (rich) context.Response.Body.
type Primer = func(ctx *gin.Context)

// EventBuilder fetches a resource by id and writes
// a json encoded event to the writer.
type EventBuilder func(id uint, method string, writer io.Writer) error

// WatchHandler handler.
type WatchHandler struct {
	BaseHandler
	Watches []*Watch
	mutex   sync.Mutex
	nextId  int
}

// Add a watch.
func (h *WatchHandler) Add(ctx *gin.Context, primer Primer, builder EventBuilder) {
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
		queue:      make(chan io.Reader, 1000),
		writer:     ctx.Writer,
		collection: collection,
		primer:     primer,
		builder:    builder,
	}
	err = h.upgrade(ctx, w)
	if err != nil {
		return
	}
	hdr := ctx.Writer.Header()
	hdr.Set("Connection", "Keep-Alive")
	ctx.Status(http.StatusOK)
	w.run()
	err = h.snapshot(ctx, afterId, w)
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
	method := ctx.Request.Method
	collection, err := h.collection(ctx, method)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	object := h.object(ctx)
	if object == nil {
		return
	}
	h.mutex.Lock()
	watches := make([]*Watch, len(h.Watches))
	copy(watches, h.Watches)
	h.mutex.Unlock()
	for _, w := range watches {
		if w.match(collection, method) {
			builder := w.builder
			if builder != nil {
				id := h.getId(object)
				if id > 0 {
					pr := h.pipedBuilder(method, id, builder)
					w.send(pr)
				}
			} else {
				pr := h.pipedEncoder(method, object)
				w.send(pr)
			}
		}
	}
}

func (h *WatchHandler) kind(object any) (kind string) {
	t := reflect.TypeOf(object)
	kind = t.Name()
	return
}

// object returns the object (response body).
func (h *WatchHandler) object(ctx *gin.Context) (object any) {
	rtx := WithContext(ctx)
	object = rtx.Response.Body
	if object == nil {
		id := h.pk(ctx)
		if id > 0 {
			object = &Ref{ID: id}
		}
	}
	return
}

// pipedEncoder json encodes the object and returns an io.Reader.
func (h *WatchHandler) pipedEncoder(method string, object any) (r io.Reader) {
	pr, pw := Pipe()
	go func() {
		encoder := json.NewEncoder(pw)
		err := encoder.Encode(
			&Event{
				Method: method,
				Object: object,
			})
		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()
	r = pr
	return
}

// pipedBuilder writes an event returns list of io.Reader.
func (h *WatchHandler) pipedBuilder(method string, id uint, builder EventBuilder) (r io.Reader) {
	pr, pw := Pipe()
	go func() {
		err := builder(id, method, pw)
		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()
	r = pr
	return
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
func (h *WatchHandler) snapshot(ctx *gin.Context, afterId uint, w *Watch) (err error) {
	if !w.match(w.collection, http.MethodPost) {
		return
	}
	rtx := WithContext(ctx)
	fake := rtx.Fake()
	if afterId > 0 {
		rtx.DB = rtx.DB.Where("id>?", afterId)
	}
	w.primer(fake)
	err = fake.Err()
	if err != nil {
		return
	}
	rtx = WithContext(fake)
	body := rtx.Response.Body
	if body == nil {
		return
	}
	method := http.MethodPost
	bt := reflect.TypeOf(body)
	switch bt.Kind() {
	case reflect.Slice:
		bv := reflect.ValueOf(body)
		for i := 0; i < bv.Len(); i++ {
			r := bv.Index(i)
			r = r.Addr()
			object := r.Interface()
			builder := w.builder
			if builder != nil {
				id := h.getId(object)
				if id > 0 {
					pr := h.pipedBuilder(method, id, builder)
					w.send(pr)
				}
			} else {
				pr := h.pipedEncoder(method, object)
				w.send(pr)
			}
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

func (h *WatchHandler) getId(object any) (id uint) {
	v := reflect.ValueOf(object)
	switch v.Kind() {
	case reflect.Pointer:
		// already
	case reflect.Struct:
		val := reflect.ValueOf(object)
		vp := reflect.New(val.Type())
		vp.Elem().Set(val)
		object = vp.Interface()
	}
	r, cast := object.(interface{ Id() uint })
	if cast {
		id = r.Id()
	}
	return
}

// PipeReader a channel-based io.Reader.
type PipeReader struct {
	input  <-chan Packet
	closed chan int
}

// Read see: io.Reader.
func (r *PipeReader) Read(b []byte) (n int, err error) {
	for n = 0; n < len(b); n++ {
		var p Packet
		p = <-r.input
		if p.err != nil {
			err = p.err
			return
		}
		if p.byte == 0 {
			if n == 0 {
				err = io.EOF
			}
			return
		}
		b[n] = p.byte
	}
	return
}

// Close the reader.
// Signals the writer.
func (r *PipeReader) Close() (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err, _ = r.(error)
		}
	}()
	close(r.closed)
	return
}

// PipeWriter is a channel-based io.Writer.
type PipeWriter struct {
	output chan<- Packet
	closed chan int
	n      int
}

// Write see: io.Writer.
// Returns an error:
// - peer is closed.
// - peer is not reading (1 day).
func (w *PipeWriter) Write(b []byte) (n int, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err, _ = r.(error)
		}
	}()
	blocked := 0
	day := 86400
	for n = 0; n < len(b); n++ {
		w.n++
		p := Packet{byte: b[n]}
		select {
		case <-w.closed:
			err = liberr.New("peer closed.")
			return
		case w.output <- p:
			blocked = 0
		case <-time.After(time.Second):
			blocked++
			if blocked > day {
				err = liberr.New("peer not reading.")
				return
			}
			n--
		}
	}
	return
}

// Close the reader.
func (w *PipeWriter) Close() (err error) {
	defer func() {
		r := recover()
		err, _ = r.(error)
	}()
	close(w.output)
	return
}

// CloseWithError close the writer.
// report send error to the reader.
func (w *PipeWriter) CloseWithError(errIn error) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err, _ = r.(error)
		}
	}()
	w.output <- Packet{err: errIn}
	err = w.Close()
	return
}

// Pipe returns a channel-based pipe.
func Pipe() (r *PipeReader, w *PipeWriter) {
	queue := make(chan Packet, 4096)
	done := make(chan int)
	r = &PipeReader{
		input:  queue,
		closed: done,
	}
	w = &PipeWriter{
		output: queue,
		closed: done,
	}
	return
}

// Packet pipe queue payload.
type Packet struct {
	byte
	err error
}
