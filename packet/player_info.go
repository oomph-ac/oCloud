package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerInfo struct {
	// ShieldID is the runtime ID of the shield item. This is required for the protocol reader/writers to
	// properly encode/decode an extra field present in the shield item.
	ShieldID int32
	// IdentityData is the Microsoft authoritative data of the player. This field may be empty if the player
	// is not authenticated with XBOX live.
	IdentityData []byte
	// ClientData is the client-authoritative data of the player.
	ClientData []byte
	// PlayerPosition is the inital position of the player.
	PlayerPosition mgl32.Vec3
}

func (*PlayerInfo) ID() uint32 {
	return IDPlayerInfo
}

func (pk *PlayerInfo) Marshal(io protocol.IO) {
	io.Int32(&pk.ShieldID)
	hasIdentityData := len(pk.IdentityData) > 0
	io.Bool(&hasIdentityData)
	if hasIdentityData {
		io.Bytes(&pk.IdentityData)
	}
	io.Bytes(&pk.ClientData)
	io.Vec3(&pk.PlayerPosition)
}
