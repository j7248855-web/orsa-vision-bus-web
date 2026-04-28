package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Broadcaster struct {
	Clients map[*websocket.Conn]bool
	Mu      sync.Mutex
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		Clients: make(map[*websocket.Conn]bool),
	}
}

func (b *Broadcaster) SendLocation(data interface{}) {
	b.Mu.Lock()
	defer b.Mu.Unlock()

	for client := range b.Clients {
		err := client.WriteJSON(data)
		if err != nil {
			client.Close()
			delete(b.Clients, client)
		}
	}
}
