package sshclip

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var evl struct {
	sync.Mutex
	listeners map[Event][]Listener
}

var (
	Interrupt = CreateUniqueEvent("Interrupt")
	Terminate = CreateUniqueEvent("Terminate")
)

func init() {
	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	loop:
		for sig := range ch {
			switch sig {
			case os.Interrupt:
				DispatchEvent(Interrupt)
			case syscall.SIGTERM:
				DispatchEvent(Terminate)
				break loop
			}
		}
	}()
}

// Event is similar to os.Signal
type Event interface {
	Event()
	String() string
}

// Listener is an event channel.
type Listener chan<- Event

// GenericEvent is a generic event.  The name is informational and has nothing
// to do with how events are dispatched.  An event with the same name can be
// created many times.  The listeners have to provide the event instance they
// want to listen to.
type GenericEvent struct {
	name string
}

func (g GenericEvent) Event() {}

func (g GenericEvent) String() string {
	return string(g.name)
}

// CreateEvent creates a GenericEvent.
func CreateEvent(name string) GenericEvent {
	return GenericEvent{name: name}
}

// CreateUniqueEvent creates a *pointer* to a GenericEvent.  The distinction
// from CreateEvent is that with a pointer, the event will be treated as a
// unique event regardless of its name.  If a listener wants to receive this
// event, it must use this instance in CreateListener.  This is useful for
// private events that shouldn't be recreated elsewhere.
func CreateUniqueEvent(name string) *GenericEvent {
	return &GenericEvent{name: name}
}

// CreateListener creates an event channel and adds events.
func CreateListener(events ...Event) chan Event {
	ch := make(chan Event)
	AddListener(ch, events...)

	return ch
}

// AddListener adds a listener to the specified events.
func AddListener(ch Listener, events ...Event) {
	evl.Lock()
	defer evl.Unlock()

	if evl.listeners == nil {
		evl.listeners = make(map[Event][]Listener)
	}

	for _, e := range events {
		listeners, ok := evl.listeners[e]
		if !ok {
			listeners = make([]Listener, 0)
		}
		evl.listeners[e] = append(listeners, ch)
	}
}

func RemoveListener(ch Listener) {
	evl.Lock()
	defer evl.Unlock()

	if evl.listeners == nil {
		return
	}

	for e, listeners := range evl.listeners {
		loopListeners := listeners
		listeners = listeners[0:0]
		for _, listener := range loopListeners {
			if listener != ch {
				listeners = append(listeners, ch)
			}
		}

		if len(listeners) == 0 {
			delete(evl.listeners, e)
		} else {
			evl.listeners[e] = listeners
		}
	}
}

// RemoveListenerEvents removes a listener from the specified events.
func RemoveListenerEvents(ch Listener, events ...Event) {
	evl.Lock()
	defer evl.Unlock()

	if evl.listeners == nil {
		return
	}

	for _, e := range events {
		if listeners, ok := evl.listeners[e]; ok {
			loopListeners := listeners
			listeners = listeners[0:0]
			for _, listener := range loopListeners {
				if listener != ch {
					listeners = append(listeners, listener)
				}
			}

			if len(listeners) == 0 {
				delete(evl.listeners, e)
			} else {
				evl.listeners[e] = listeners
			}
		}
	}
}

// DispatchEvent dispatches an event to all listeners.
func DispatchEvent(event Event) {
	evl.Lock()
	defer evl.Unlock()

	if evl.listeners == nil {
		return
	}

	if listeners, ok := evl.listeners[event]; ok {
		for _, ch := range listeners {
			ch <- event
		}
	}
}
