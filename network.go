package main

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/oomph-ac/ocloud/client"
	"github.com/oomph-ac/ocloud/client/handler"
	"github.com/quic-go/quic-go"
)

func handleConn(conn quic.Connection) {
	defer func() {
		if v := recover(); v != nil {
			hub := sentry.CurrentHub().Clone()
			hub.Scope().SetTag("context", "connection")
			hub.Scope().SetTag("addr", conn.RemoteAddr().String())
			_ = hub.Recover(v)
			_ = hub.Flush(time.Second * 5)
		}
	}()

	// Start listening and accepting streams from the connection.
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			logger.Error().
				Err(err).
				Str("addr", conn.RemoteAddr().String()).
				Msg("failed to accept stream")
			return
		}

		c := client.New(stream, conn.RemoteAddr(), logger)
		c.RegisterHandlers(handler.NewAuthenticationHandler(c), handler.NewOomphRecorder(c))

		// TODO: Should we be storing this client somewhere?
	}
}

func listen(l *quic.Listener) {
	defer func() {
		if v := recover(); v != nil {
			hub := sentry.CurrentHub().Clone()
			hub.Scope().SetTag("context", "listener")
			_ = hub.Recover(v)
			_ = hub.Flush(time.Second * 5)
		}
	}()

	ctx := context.Background()
	for {
		if conn, err := l.Accept(ctx); err == nil {
			go handleConn(conn)
		} else {
			logger.Error().Err(err).Msg("failed to accept connection")
		}
	}
}
