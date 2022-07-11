package processor

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastrucure"
)

type HandlingMiddleware interface {
	Wrap(next infrastrucure.HandlerFunc) infrastrucure.HandlerFunc
}

type RequestProcessor struct {
	handlerFunc infrastrucure.HandlerFunc
}

func NewRequestProcessor(options ...requestProcessorOption) *RequestProcessor {
	processor := &RequestProcessor{handlerFunc: finalFandler}

	for i := len(options) - 1; i >= 0; i-- {
		option := options[i]
		option(processor)
	}

	return processor
}

func (rp *RequestProcessor) HandleRequest(w http.ResponseWriter, req *http.Request) {
	rp.handlerFunc(w, req)
}

func finalFandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	fmt.Fprintln(w, "Failed requset handler")
}