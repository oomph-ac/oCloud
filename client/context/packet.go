package context

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// PacketContext is a context used for handlers processing packets recieved from the underlying client.
type PacketContext struct {
	pk        packet.Packet
	err       error
	cancelled bool

	completed bool
}

func NewPacketCtx(pk packet.Packet) *PacketContext {
	return &PacketContext{
		pk: pk,
	}
}

func (ctx *PacketContext) Cancel() {
	if ctx.completed {
		panic("cannot use completed packet context")
	}
	ctx.cancelled = true
}

func (ctx *PacketContext) Cancelled() bool {
	if ctx.completed {
		panic("cannot use completed packet context")
	}
	return ctx.cancelled
}

func (ctx *PacketContext) Error() error {
	if ctx.completed {
		panic("cannot use completed packet context")
	}
	return ctx.err
}

func (ctx *PacketContext) SetError(err error) {
	if ctx.completed {
		panic("cannot use completed packet context")
	} else if ctx.err == nil {
		ctx.err = err
	}
}

func (ctx *PacketContext) Packet() packet.Packet {
	if ctx.completed {
		panic("cannot use completed packet context")
	}
	return ctx.pk
}

func (ctx *PacketContext) Done() {
	if ctx.completed {
		panic("cannot use completed packet context")
	}

	ctx.pk = nil
	ctx.err = nil
	ctx.completed = true
}
