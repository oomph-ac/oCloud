package client

import (
	"bytes"
	"compress/zlib"
	"net"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	// ClientReadModePacketLength is the read mode where the client is reading the length of the packet.
	ClientReadModePacketLength byte = iota
	// ClientReadModePacketData is the read mode where the client is reading the data of a packet batch.
	// This is used after the client had read the expected length of the packet.
	ClientReadModePacketData
)

type Client struct {
	conn quic.Stream
	addr net.Addr

	log zerolog.Logger

	compressor *zlib.Writer

	// rBuffer is the buffer that is used to read packets from the underlying connection. Specifically, this buffer
	// contains the de-compressed data from the underlying connection. Only one goroutine accesses this buffer, so
	// we don't need to use a mutex to protect it.
	rBuffer *bytes.Buffer

	// wBuffer is the buffer that is used to write packets to the underlying connection. Specifically, this buffer
	// contains the non-compressed data. When the ticker is ran, the data is compressed and written to the underlying
	// connection.
	wBuffer *bytes.Buffer
	// writePks is the number of packets that are expected to be written to the underlying connection. This is used when
	// writing the final batch of packets to the underlying connection. It is incremented whenever Write() is called on the client.
	writePks uint64
	writeMu  sync.Mutex

	protoReader atomic.Pointer[protocol.Reader]
	protoWriter atomic.Pointer[protocol.Writer]

	handlers map[uuid.UUID]PacketHandler
	hMu      sync.RWMutex

	// deferredPackets is a channel that is used when there are no handlers registered to this client.
	// It will be read from when at least one handler is registered.
	deferredPackets chan packet.Packet
	close           chan struct{}
	onceClose       sync.Once

	authenticated atomic.Bool
	connected     atomic.Bool
}

func New(
	conn quic.Stream,
	addr net.Addr,
	log zerolog.Logger,
) *Client {
	c := &Client{
		conn: conn,
		addr: addr,

		log: log,

		rBuffer: bytes.NewBuffer(make([]byte, 10*1024*1024)),
		wBuffer: bytes.NewBuffer(make([]byte, 0, 65535)),

		close:           make(chan struct{}, 1),
		deferredPackets: make(chan packet.Packet, 65535),
	}

	c.compressor, _ = zlib.NewWriterLevel(conn, 7)
	c.connected.Store(true)

	// The shield ID is set to zero for now, until the client sends a ClientInfo packet which specifies what the shield ID is.
	c.protoReader.Store(protocol.NewReader(c.rBuffer, 0, false))
	c.protoWriter.Store(protocol.NewWriter(c.wBuffer, 0))

	go c.startTicking()
	return c
}

func (c *Client) Authenticated() bool {
	return c.authenticated.Load()
}

func (c *Client) SetAuthenticated(authenticated bool) {
	c.authenticated.Store(authenticated)
}

// Address returns the network address of the client.
func (c *Client) Addr() net.Addr {
	return c.addr
}

// Reader returns the protocol reader of the client. This is used to read packets from the underlying connection.
func (c *Client) Reader() *protocol.Reader {
	return c.protoReader.Load()
}

// SetReader sets the protocol reader of the client. This is used to read packets from the underlying connection.
func (c *Client) SetReader(reader *protocol.Reader) {
	c.protoReader.Store(reader)
}

// Writer returns the protocol writer of the client. This is used to write packets to the underlying connection.`
func (c *Client) Writer() *protocol.Writer {
	return c.protoWriter.Load()
}

// SetWriter sets the protocol writer of the client. This is used to write packets to the underlying connection.
func (c *Client) SetWriter(writer *protocol.Writer) {
	c.protoWriter.Store(writer)
}

// Close closes the stream to the underlying stream. An error is returned if the close fails.
func (c *Client) Close(err error) (closeErr error) {
	c.onceClose.Do(func() {
		if err != nil {
			c.log.Error().
				Err(err).
				Str("addr", c.addr.String()).
				Msg("client closed")
		}

		c.connected.Store(false)

		close(c.deferredPackets)
		close(c.close)

		c.compressor.Close()
		c.rBuffer = nil

		c.hMu.Lock()
		for _, handler := range c.handlers {
			_ = handler.Close()
		}
		c.handlers = nil
		c.hMu.Unlock()

		closeErr = c.conn.Close()
	})
	return
}
