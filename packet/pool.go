package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

const (
	IDAuthenticate uint32 = iota
	IDPlayerInfo
)

var pool = make(map[uint32]func() packet.Packet)

func init() {
	Register(func() packet.Packet { return &Authenticate{} })
}

func Register(pkFunc func() packet.Packet) {
	pk := pkFunc()
	pool[pk.ID()] = pkFunc
}

func Find(id uint32) packet.Packet {
	if f, ok := pool[id]; ok {
		return f()
	}
	return nil
}
