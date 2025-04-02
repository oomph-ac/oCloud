package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

// Authenticate is a packet sent by a client to the Oomph cloud to authenticate. The JWT token
// provided in this case should be one retrieved from the Oomph API.
type Authenticate struct {
	// Token is the JWT token used to validate the connection to the Oomph cloud.
	Token string
}

func (*Authenticate) ID() uint32 {
	return IDAuthenticate
}

func (pk *Authenticate) Marshal(io protocol.IO) {
	io.String(&pk.Token)
}
