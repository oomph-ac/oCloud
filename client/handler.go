package client

import (
	"maps"

	"github.com/google/uuid"
	"github.com/oomph-ac/ocloud/client/context"
)

// PacketHandler is an interface used to handle incoming packets from the client.
type PacketHandler interface {
	// SetID sets the UUID identifier of the packet handler.
	SetID(id uuid.UUID)

	// Recieve handles a packet from the underlying connection.
	Recieve(ctx *context.PacketContext)
	// Close closes the packet handler and releases any resources it holds.
	Close() error
}

// RegisterHandlers registers multiple handlers at once with the client.
func (c *Client) RegisterHandlers(handlers ...PacketHandler) {
	c.hMu.Lock()
	defer c.hMu.Unlock()

	for _, handler := range handlers {
		randUuid, _ := uuid.NewRandom()
		handler.SetID(randUuid)
		c.handlers[randUuid] = handler
	}
}

// RegisterHandler registers a packet handler with the client. It returns a UUID that can be used to later unregister
// the handler.
func (c *Client) RegisterHandler(handler PacketHandler) {
	c.hMu.Lock()
	defer c.hMu.Unlock()

	randUuid, _ := uuid.NewRandom()
	handler.SetID(randUuid)
	c.handlers[randUuid] = handler
}

// UnregisterHandler unregisters a packet handler with the client.
func (c *Client) UnregisterHandler(uuid uuid.UUID) {
	c.hMu.Lock()
	defer c.hMu.Unlock()

	if h, ok := c.handlers[uuid]; ok {
		_ = h.Close()
		delete(c.handlers, uuid)
	}
}

// Handlers returns a cloned map of the handlers registered with the client.
func (c *Client) Handlers() map[uuid.UUID]PacketHandler {
	c.hMu.RLock()
	defer c.hMu.RUnlock()

	return maps.Clone(c.handlers)
}
