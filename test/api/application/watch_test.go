package application

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/binding"
	"github.com/konveyor/tackle2-hub/test/assert"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"
)

type TestEventHandler struct {
	events []*api.Event
	errors []error
}

func (h *TestEventHandler) Event(event *api.Event) {
	h.events = append(h.events, event)
}

func (h *TestEventHandler) Error(err error) {
	h.errors = append(h.errors, err)
}

func TestApplicationWatch(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	nEvent := 0
	begin := []*api.Application{
		{
			Name: fmt.Sprintf("Test-%d", rand.Int()),
		},
		{
			Name: fmt.Sprintf("Test-%d", rand.Int()),
		},
		{
			Name: fmt.Sprintf("Test-%d", rand.Int()),
		},
	}
	for _, r := range begin {
		err := Application.Create(r)
		g.Expect(err).To(gomega.BeNil())
		nEvent++
	}

	mark := time.Now()
	mark = mark.Add(time.Second * 3)
	handler := &TestEventHandler{}
	ctx := context.Background()
	ctx, cancel := context.WithDeadline(ctx, mark)
	defer func() {
		cancel()
	}()

	assert.Must(t, Application.Watch(
		ctx,
		handler,
		&binding.WatchOptions{
			AfterId: 1,
			Methods: []string{http.MethodPost},
		}))

	add := []*api.Application{
		{
			Name: fmt.Sprintf("Test-%d", rand.Int()),
		},
		{
			Name: fmt.Sprintf("Test-%d", rand.Int()),
		},
		{
			Name: fmt.Sprintf("Test-%d", rand.Int()),
		},
	}
	for _, r := range add {
		err := Application.Create(r)
		g.Expect(err).To(gomega.BeNil())
	}

	expected := []*api.Event{
		{
			Method: http.MethodPost,
			Kind:   "application",
			ID:     2,
		},
		{
			Method: http.MethodPost,
			Kind:   "application",
			ID:     3,
		},
		{
			Method: http.MethodPost,
			Kind:   "application",
			ID:     4,
		},
		{
			Method: http.MethodPost,
			Kind:   "application",
			ID:     5,
		},
		{
			Method: http.MethodPost,
			Kind:   "application",
			ID:     6,
		},
	}
	g.Expect(len(handler.errors)).To(gomega.Equal(0))
	g.Expect(expected).To(gomega.Equal(handler.events))

	for _, r := range append(begin, add...) {
		err := Application.Delete(r.ID)
		g.Expect(err).To(gomega.BeNil())
	}
}
