package streamlog

import (
	"expvar"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
)

var droppedMessages = expvar.NewMap("streamlog-dropped-messages")

// A StreamLogger makes messages sent to it available through HTTP.
type StreamLogger struct {
	dataQueue  chan Formatter
	subscribed map[io.Writer]subscription
	url        string
	mu         sync.Mutex
	// size is used to check if there are any subscriptions. Keep
	// it atomically in sync with the size of subscribed.
	size uint32
	// seq is a guard for modifications of subscribed - increment
	// it atomically whenever you modify it.
	seq uint32
}

type subscription struct {
	done   chan bool
	params url.Values
}

type Formatter interface {
	Format(url.Values) string
}

func (logger *StreamLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		// FIXME(szopa): send a malformed request error.
	}
	<-logger.subscribe(w, r.Form)
}

func (logger *StreamLogger) subscribe(w io.Writer, params url.Values) chan bool {
	done := make(chan bool)
	logger.mu.Lock()
	defer logger.mu.Unlock()

	logger.subscribed[w] = subscription{done: done, params: params}
	atomic.AddUint32(&logger.seq, 1)
	atomic.StoreUint32(&logger.size, uint32(len(logger.subscribed)))
	return done
}

// New returns a new StreamLogger with a buffer that can contain size
// messages. Any messages sent to it will be available at url.
func New(url string, size int) *StreamLogger {
	logger := &StreamLogger{
		dataQueue:  make(chan Formatter, size),
		subscribed: make(map[io.Writer]subscription),
		url:        url,
	}
	go logger.stream()
	http.Handle(url, logger)
	return logger
}

// stream sends messages sent to logger to all of its subscribed
// writers. This method should be called in a goroutine.
func (logger *StreamLogger) stream() {
	seq := uint32(0)
	var subscribed map[io.Writer]subscription

	for message := range logger.dataQueue {

		if s := atomic.LoadUint32(&(logger.seq)); s != seq {
			logger.mu.Lock()
			subscribed = make(map[io.Writer]subscription, len(logger.subscribed))
			for w, subscription := range logger.subscribed {
				subscribed[w] = subscription
			}
			seq = atomic.LoadUint32(&(logger.seq))
			logger.mu.Unlock()
		}

		if len(subscribed) == 0 {
			continue
		}

		for w, subscription := range subscribed {
			messageString := message.Format(subscription.params) + "\n"
			if _, err := io.WriteString(w, messageString); err != nil {
				subscription.done <- true

				logger.mu.Lock()
				delete(logger.subscribed, w)
				atomic.AddUint32(&logger.seq, 1)
				atomic.StoreUint32(&logger.size, uint32(len(logger.subscribed)))
				logger.mu.Unlock()
			} else {
				w.(http.Flusher).Flush()
			}
		}
	}
}

// Send sends message to all the writers subscribed to logger. Calling
// Send does not block.
func (logger *StreamLogger) Send(message Formatter) {
	if atomic.LoadUint32(&logger.size) == 0 {
		// There are no subscribers, do nothing.
		return
	}
	select {
	case logger.dataQueue <- message:
	default:
		droppedMessages.Add(logger.url, 1)
	}
}
