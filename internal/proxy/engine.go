package proxy

import (
	"context"
	"log/slog"
	"net"

	"github.com/alfred-identity/app/internal/protocol"
	"github.com/alfred-identity/app/internal/router"
)

// Engine implements the p99-login-proxy UDP relay with SOE CRC and session sequencing.
type Engine struct {
	Router  *router.Router
	Session protocol.ProxySessionState
	Log     *slog.Logger

	crcBytes byte
	crcKey   uint32
	maxPacket uint32
	client   *net.UDPAddr
}

// Actions are outbound datagrams before CRC is restored.
type Actions struct {
	SendClient   [][]byte
	SendUpstream [][]byte
}

func (e *Engine) ClientAddr() *net.UDPAddr {
	return e.client
}

func (e *Engine) OnDatagram(ctx context.Context, data []byte, from, upstream *net.UDPAddr) Actions {
	packet := e.stripCRC(data)
	if isUpstreamPeer(from, upstream) {
		return e.handleServer(ctx, packet)
	}
	e.client = from
	return e.handleClient(ctx, packet)
}

// Finalize restores SOE CRC on wire-bound packets.
func (e *Engine) Finalize(packets [][]byte) [][]byte {
	if e.crcBytes == 0 {
		return packets
	}
	out := make([][]byte, len(packets))
	for i, p := range packets {
		if protocol.PacketUsesCRC(p) {
			out[i] = protocol.AppendCRC(p, e.crcKey, e.crcBytes)
		} else {
			out[i] = append([]byte{}, p...)
		}
	}
	return out
}

func (e *Engine) stripCRC(data []byte) []byte {
	if e.crcBytes == 0 || !protocol.PacketUsesCRC(data) {
		return append([]byte{}, data...)
	}
	return append([]byte{}, protocol.StripCRC(data, e.crcBytes)...)
}

func (e *Engine) handleClient(ctx context.Context, data []byte) Actions {
	var actions Actions
	outbound := append([]byte{}, data...)
	op := protocol.TransportOpcode(outbound)

	switch op {
	case protocol.OpCombined:
		e.Session.AdjustCombined(outbound)
		if login, ok := protocol.ParseLoginPacket(outbound); ok && e.Router != nil {
			res := e.Router.HandleLoginPacket(ctx, login)
			switch res.Decision {
			case router.DecisionFail:
				if e.Log != nil {
					e.Log.Warn("login rewrite failed; not forwarding",
						"msg", res.Message, "user", login.Username)
				}
				return actions
			default:
				outbound = res.Packet
				if e.Log != nil {
					e.Log.Info("login routed", "decision", string(res.Decision), "user", login.Username)
				}
			}
		}
	case protocol.OpAck:
		e.Session.AdjustAck(outbound, 0)
	case protocol.OpPacket:
		e.Session.AdjustClientPacket(outbound, 0)
	case protocol.OpFragment:
		e.Session.AdjustClientPacket(outbound, 0)
	}

	if len(outbound) > 0 {
		actions.SendUpstream = append(actions.SendUpstream, outbound)
	}
	return actions
}

func (e *Engine) handleServer(ctx context.Context, data []byte) Actions {
	var actions Actions
	if len(data) == 0 {
		return actions
	}
	op := protocol.TransportOpcode(data)

	switch op {
	case protocol.OpSessionResponse:
		if resp, ok := protocol.ParseSessionResponse(data); ok {
			e.crcBytes = resp.CRCBytes
			e.crcKey = resp.EncodeKey
			e.maxPacket = resp.MaxPacketSize
			e.Session.Reset()
			if e.Log != nil {
				e.Log.Info("login server session parameters",
					"crc_bytes", resp.CRCBytes, "crc_key", resp.EncodeKey,
					"max_packet", resp.MaxPacketSize)
			}
		}
		actions.SendClient = append(actions.SendClient, append([]byte{}, data...))
	case protocol.OpFragment:
		out := e.Session.RecvFragment(append([]byte{}, data...), int(e.maxPacket))
		if len(out) > 0 && e.Log != nil {
			e.Log.Info("server list forwarded to client", "packets", len(out))
		}
		actions.SendClient = append(actions.SendClient, out...)
	case protocol.OpCombined:
		buf := append([]byte{}, data...)
		forward := e.Session.RecvCombined(buf, 0, len(buf))
		actions.SendClient = append(actions.SendClient, forward)
	case protocol.OpPacket:
		buf := append([]byte{}, data...)
		e.Session.RecvPacket(buf, 0)
		actions.SendClient = append(actions.SendClient, buf)
	case protocol.OpAck:
		buf := append([]byte{}, data...)
		e.Session.AdjustServerAck(buf, 0)
		actions.SendClient = append(actions.SendClient, buf)
	default:
		actions.SendClient = append(actions.SendClient, append([]byte{}, data...))
	}
	return actions
}
