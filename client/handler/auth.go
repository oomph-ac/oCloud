package handler

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/oomph-ac/ocloud/client"
	"github.com/oomph-ac/ocloud/client/context"
	"github.com/oomph-ac/ocloud/client/jwt"
	cloudpacket "github.com/oomph-ac/ocloud/packet"
)

type AuthenticationHandler struct {
	mClient *client.Client
	id      uuid.UUID
}

func NewAuthenticationHandler(c *client.Client) *AuthenticationHandler {
	return &AuthenticationHandler{mClient: c}
}

func (h *AuthenticationHandler) SetID(id uuid.UUID) {
	h.id = id
}

func (h *AuthenticationHandler) Recieve(ctx *context.PacketContext) {
	// If the client is already authenticated, this handler is no longer required.
	c := h.mClient
	if c.Authenticated() {
		return
	}

	pk, ok := ctx.Packet().(*cloudpacket.Authenticate)
	if !ok {
		ctx.SetError(fmt.Errorf("expected authentication packet, got %T", ctx.Packet()))
		return
	}
	if _, ok := jwt.Validate(pk.Token); !ok {
		ctx.SetError(fmt.Errorf("unable to validate authentication token"))
		return
	}

	// Now that we are authenticated, we can remove this handler from the client and set the client to
	// authenticated. We use a goroutine to unregister the handler to avoid a deadlock.
	go c.UnregisterHandler(h.id)
	c.SetAuthenticated(true)
}

func (h *AuthenticationHandler) Close() error {
	h.mClient = nil
	return nil
}
