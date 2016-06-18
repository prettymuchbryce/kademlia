package dht

import "sync"

type broadcast struct {
	listeners map[string]*topic
	mutex     *sync.Mutex
}

type topic struct {
	ch map[int]chan (interface{})
	co int
}

func (b *broadcast) init() {
	b.listeners = make(map[string]*topic)
	b.mutex = &sync.Mutex{}
}

func (b *broadcast) addListener(s string) (int, chan (interface{})) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	if b.listeners[s] == nil {
		b.listeners[s] = &topic{}
		b.listeners[s].ch = make(map[int]chan (interface{}), 0)
	}
	listener := make(chan (interface{}))
	b.listeners[s].ch[b.listeners[s].co] = listener

	defer func() {
		b.listeners[s].co++
	}()

	return b.listeners[s].co, listener
}

func (b *broadcast) removeAll() {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	for s := range b.listeners {
		if b.listeners[s] == nil {
			continue
		}

		for _, l := range b.listeners[s].ch {
			close(l)
		}

		delete(b.listeners, s)
	}
}

func (b *broadcast) removeListener(id int, s string) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	if b.listeners[s] == nil || b.listeners[s].ch[id] == nil {
		return
	}

	close(b.listeners[s].ch[id])
	delete(b.listeners[s].ch, id)

	if len(b.listeners[s].ch) == 0 {
		delete(b.listeners, s)
	}
}

func (b *broadcast) dispatch(s string, value interface{}) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	if b.listeners[s] == nil {
		return
	}

	for _, l := range b.listeners[s].ch {
		l <- value
	}
}
