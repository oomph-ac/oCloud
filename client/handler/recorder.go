package handler

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/oomph-ac/ocloud/client"
	"github.com/oomph-ac/ocloud/client/context"
	cloudpacket "github.com/oomph-ac/ocloud/packet"
)

// OomphRecorder is a packet handler that records packets related to Oomph events.
// This allows for player sessions to be recorded and replayed in the future for
// potential debug or general analysis purposes.
type OomphRecorder struct {
	mClient *client.Client
	id      uuid.UUID
}

func NewOomphRecorder(c *client.Client) *OomphRecorder {
	return &OomphRecorder{mClient: c}
}

func (r *OomphRecorder) SetID(id uuid.UUID) {
	r.id = id
}

func (r *OomphRecorder) Recieve(ctx *context.PacketContext) {
	if !r.mClient.Authenticated() {
		ctx.SetError(fmt.Errorf("client not authenticated"))
		return
	}

	// TODO: Implementation of recording player/Oomph events.
	switch pk := ctx.Packet().(type) {
	case *cloudpacket.Authenticate:
		fmt.Println(pk)
	}
}

func (r *OomphRecorder) Close() error {
	r.mClient = nil
	return nil
}
