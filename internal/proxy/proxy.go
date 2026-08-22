package proxy

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/alfred-identity/app/internal/protocol"
	"github.com/alfred-identity/app/internal/router"
)

// Server is a UDP middleman between EQ client and the login server.
type Server struct {
	Listen   string
	Upstream string
	Router   *router.Router
	Log      *slog.Logger

	conn   *net.UDPConn
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (s *Server) Start(parent context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", s.Listen)
	if err != nil {
		return err
	}
	c, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err // port in use → fail
	}
	upAddr, err := net.ResolveUDPAddr("udp", s.Upstream)
	if err != nil {
		_ = c.Close()
		return err
	}

	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.conn = c
	s.cancel = cancel
	s.mu.Unlock()

	log := s.Log
	if log == nil {
		log = slog.Default()
	}

	go func() {
		defer c.Close()
		buf := make([]byte, 65535)
		var clientAddr *net.UDPAddr
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, peer, err := c.ReadFromUDP(buf)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			pkt := append([]byte{}, buf[:n]...)

			if isUpstreamPeer(peer, upAddr) {
				if clientAddr == nil {
					log.Debug("dropping upstream packet: no EQ client bound")
					continue
				}
				if _, err := c.WriteToUDP(pkt, clientAddr); err != nil {
					log.Warn("send to client failed", "err", err)
				}
				continue
			}

			clientAddr = peer
			out := pkt
			if isCombinedLogin(pkt) {
				user := extractUsernameHint(pkt)
				if user != "" && s.Router != nil {
					res := s.Router.HandleLoginPacket(ctx, pkt, user)
					switch res.Decision {
					case router.DecisionFail:
						log.Warn("login rewrite failed; forwarding original", "msg", res.Message, "user", user)
						out = pkt
					default:
						out = res.Packet
						log.Info("login routed", "decision", string(res.Decision), "user", user)
					}
				}
			}

			if _, err := c.WriteToUDP(out, upAddr); err != nil {
				log.Warn("send to upstream failed", "err", err)
			}
		}
	}()
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

func isUpstreamPeer(peer, upstream *net.UDPAddr) bool {
	if peer == nil || upstream == nil {
		return false
	}
	return peer.IP.Equal(upstream.IP) && peer.Port == upstream.Port
}

func isCombinedLogin(pkt []byte) bool {
	if len(pkt) < 2 || pkt[0] != 0x00 || pkt[1] != 0x03 {
		return false
	}
	_, _, ok := protocol.FindLoginCipherOffset(pkt)
	return ok
}

func extractUsernameHint(pkt []byte) string {
	start, _, ok := protocol.FindLoginCipherOffset(pkt)
	if !ok {
		return ""
	}
	ct := pkt[start:]
	if len(ct) == 0 || len(ct)%8 != 0 {
		return ""
	}
	pt, err := protocol.DecryptDES(ct)
	if err != nil {
		return ""
	}
	for i, b := range pt {
		if b == 0 {
			return string(pt[:i])
		}
	}
	return ""
}
