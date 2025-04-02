package client

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"time"

	"github.com/oomph-ac/ocloud/client/context"
	cloudpacket "github.com/oomph-ac/ocloud/packet"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (c *Client) Write(pk packet.Packet) (err error) {
	if !c.connected.Load() {
		return fmt.Errorf("client not connected")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	protoWriter := c.protoWriter.Load()
	if protoWriter == nil {
		return fmt.Errorf("protoWriter is nil")
	}

	defer func() {
		if v := recover(); v != nil {
			c.log.Error().
				Err(fmt.Errorf("%v", v)).
				Uint32("packet_id", pk.ID()).
				Msg("client crashed while writing packet")
		}
	}()

	// The protocol writer is directly linked to the wBuffer of the client.
	pk.Marshal(protoWriter)
	c.writePks++

	return nil
}

// Flush writes all pending packets to the underlying connection. An error is returned if the write to the
// underlying connection fails.
func (c *Client) Flush() error {
	if !c.connected.Load() {
		return fmt.Errorf("client not connected")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.writePks == 0 {
		return nil
	}

	// Write the header (length and packet count) directly to the underlying connection.
	header := make([]byte, cloudpacket.HeaderSize)
	header[0], header[1], header[2], header[3] = byte(c.wBuffer.Len()), byte(c.wBuffer.Len()>>8), byte(c.wBuffer.Len()>>16), byte(c.wBuffer.Len()>>24)
	header[4], header[5], header[6], header[7], header[8], header[9], header[10], header[11] = byte(c.writePks), byte(c.writePks>>8), byte(c.writePks>>16), byte(c.writePks>>24), byte(c.writePks>>32), byte(c.writePks>>40), byte(c.writePks>>48), byte(c.writePks>>56)

	// Write the header to the underlying connection.
	if _, err := c.conn.Write(header); err != nil {
		c.connected.Store(false)
		return fmt.Errorf("failed to write header: %v", err)
	}

	// Write the compressed data to the underlying connection.
	if _, err := c.compressor.Write(c.wBuffer.Bytes()); err != nil {
		c.connected.Store(false)
		return fmt.Errorf("failed to write compressed data: %v", err)
	}

	// Flush the compressor to ensure all data is written to the underlying connection.
	if err := c.compressor.Flush(); err != nil {
		c.connected.Store(false)
		return fmt.Errorf("failed to flush compressed data: %v", err)
	}

	c.writePks = 0
	c.wBuffer.Reset()
	return nil
}

func (c *Client) handleDeferred() {
	c.hMu.RLock()
	defer c.hMu.RUnlock()

	if len(c.handlers) == 0 {
		return
	}

	for {
		select {
		case pk := <-c.deferredPackets:
			ctx := context.NewPacketCtx(pk)
			for _, handler := range c.handlers {
				handler.Recieve(ctx)
			}
		default:
			return
		}
	}
}

// startTicking starts the read/write loop for the client. It reads packets from the underlying connection,
// and sends them to the pending packets channel.
func (c *Client) startTicking() {
	defer func() {
		if v := recover(); v != nil {
			c.Close(fmt.Errorf("crashed while reading from client: %v", v))
			return
		}
		c.Close(nil)
	}()

	var (
		readLength int = cloudpacket.HeaderSize
		readPks    int = 0

		readBuffer     = bytes.NewBuffer(make([]byte, cloudpacket.MaxExpectedPacketSize))
		batchReader, _ = zlib.NewReader(readBuffer)

		readingHeader bool = true

		err error
	)

	t := time.NewTicker(time.Millisecond * 500)
	defer t.Stop()

	for {
		select {
		case <-c.close:
			return
		case <-t.C:
			// Flush the connection to write any packets we need to send to the client.
			if err := c.Flush(); err != nil {
				c.Close(fmt.Errorf("failed to flush client: %v", err))
				return
			}
		default:
			// Read data from the underlying connection and write it to the buffer.
			readBuf := readBuffer.Bytes()[0:readLength]
			if err = c.readFromConnection(readBuf); err != nil {
				return
			}

			if readingHeader {
				readLength, readPks, err = c.parseHeader(readBuf)
				if err != nil {
					return
				}
				readingHeader = false
			} else {
				if err = c.processBatch(batchReader, readBuffer, readLength, readPks); err != nil {
					return
				}

				// Reset the read mode to read the next packet length.
				readLength = cloudpacket.HeaderSize
				readingHeader = true
				readPks = 0
			}
		}
	}
}

// readFromConnection reads data from the connection into the provided buffer.
func (c *Client) readFromConnection(buf []byte) error {
	_, err := io.ReadFull(c.conn, buf)
	if err != nil && c.connected.Load() {
		c.Close(fmt.Errorf("failed to read from connection: %v", err))
		return err
	}
	return nil
}

// parseHeader parses the packet header and returns the batch length and packet count.
func (c *Client) parseHeader(buf []byte) (_ int, _ int, err error) {
	// Read a little-endian unsigned 32-bit integer from the buffer.
	batchLength := int(buf[0]) |
		int(buf[1])<<8 |
		int(buf[2])<<16 |
		int(buf[3])<<24
	if batchLength > cap(buf) || batchLength < 0 {
		err = fmt.Errorf("invalid packet length: %d", batchLength)
		c.Close(err)
		return
	}

	packetCount := int(buf[4]) |
		int(buf[5])<<8 |
		int(buf[6])<<16 |
		int(buf[7])<<24 |
		int(buf[8])<<32 |
		int(buf[9])<<40 |
		int(buf[10])<<48 |
		int(buf[11])<<56
	if packetCount <= 0 {
		err = fmt.Errorf("invalid packet count: %d", packetCount)
		c.Close(err)
		return
	}
	return batchLength, packetCount, nil
}

// processBatch processes a batch of packets from the decompressed data.
func (c *Client) processBatch(batchReader io.Reader, readBuffer *bytes.Buffer, readLength int, packetCount int) error {
	_, err := batchReader.Read(c.rBuffer.Bytes()[0:readLength])
	if err != nil {
		c.Close(fmt.Errorf("failed to decompress batch: %v", err))
		return err
	}

	// Process each packet in the batch
	for i := 0; i < packetCount; i++ {
		if err := c.processPacket(); err != nil {
			return err
		}
	}

	return nil
}

// processPacket processes a single packet from the batch.
func (c *Client) processPacket() (err error) {
	var (
		protoReader = c.protoReader.Load()
		packetId    uint32
	)
	if protoReader == nil {
		err = fmt.Errorf("protoReader is nil")
		c.Close(err)
		return
	}
	protoReader.Uint32(&packetId)

	pk := cloudpacket.Find(packetId)
	if pk == nil {
		err = fmt.Errorf("unknown packet ID %d", packetId)
		c.Close(err)
		return
	}
	pk.Marshal(protoReader)

	// Check to see if the client has been closed first before allowing handlers to be called.
	select {
	case <-c.close:
		return fmt.Errorf("client closed")
	default:
		return c.handlePacket(pk)
	}
}

// handlePacket processes a packet through the appropriate handlers.
func (c *Client) handlePacket(pk packet.Packet) error {
	// Handle any deferred packets that are waiting to be processed by the handlers.
	c.handleDeferred()

	c.hMu.RLock()
	defer c.hMu.RUnlock()

	if len(c.handlers) == 0 {
		c.deferredPackets <- pk
		return nil
	}

	ctx := context.NewPacketCtx(pk)
	defer ctx.Done()

	for _, handler := range c.handlers {
		handler.Recieve(ctx)
	}

	if err := ctx.Error(); err != nil {
		err = fmt.Errorf("error while processing %T: %v", pk, err)
		c.Close(err)
		return err
	}
	return nil
}
