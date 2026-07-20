package counter

import (
	"io"
	"net"

	"github.com/sagernet/sing/common/bufio"

	"github.com/sagernet/sing/common/buf"

	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
)

type ConnCounter struct {
	network.ExtendedConn
	storage   *TrafficStorage
	readFunc  network.CountFunc
	writeFunc network.CountFunc
}

func NewConnCounter(conn net.Conn, s *TrafficStorage) net.Conn {
	return &ConnCounter{
		ExtendedConn: bufio.NewExtendedConn(conn),
		storage:      s,
		readFunc: func(n int64) {
			s.UpCounter.Add(n)
		},
		writeFunc: func(n int64) {
			s.DownCounter.Add(n)
		},
	}
}

func (c *ConnCounter) Read(b []byte) (n int, err error) {
	return c.ExtendedConn.Read(b)
}

func (c *ConnCounter) Write(b []byte) (n int, err error) {
	return c.ExtendedConn.Write(b)
}

func (c *ConnCounter) ReadBuffer(buffer *buf.Buffer) error {
	return c.ExtendedConn.ReadBuffer(buffer)
}

func (c *ConnCounter) WriteBuffer(buffer *buf.Buffer) error {
	return c.ExtendedConn.WriteBuffer(buffer)
}

func (c *ConnCounter) UnwrapReader() (io.Reader, []network.CountFunc) {
	return c.ExtendedConn, []network.CountFunc{
		c.readFunc,
	}
}

func (c *ConnCounter) UnwrapWriter() (io.Writer, []network.CountFunc) {
	return c.ExtendedConn, []network.CountFunc{
		c.writeFunc,
	}
}

func (c *ConnCounter) Upstream() any {
	return c.ExtendedConn
}

type PacketConnCounter struct {
	network.PacketConn
	storage   *TrafficStorage
	readFunc  network.CountFunc
	writeFunc network.CountFunc
}

func NewPacketConnCounter(conn network.PacketConn, s *TrafficStorage) network.PacketConn {
	return &PacketConnCounter{
		PacketConn: conn,
		storage:    s,
		readFunc: func(n int64) {
			s.UpCounter.Add(n)
		},
		writeFunc: func(n int64) {
			s.DownCounter.Add(n)
		},
	}
}

func (p *PacketConnCounter) ReadPacket(buff *buf.Buffer) (destination M.Socksaddr, err error) {
	return p.PacketConn.ReadPacket(buff)
}

func (p *PacketConnCounter) WritePacket(buff *buf.Buffer, destination M.Socksaddr) (err error) {
	return p.PacketConn.WritePacket(buff, destination)
}

func (p *PacketConnCounter) UnwrapPacketReader() (network.PacketReader, []network.CountFunc) {
	return p.PacketConn, []network.CountFunc{
		p.readFunc,
	}
}

func (p *PacketConnCounter) UnwrapPacketWriter() (network.PacketWriter, []network.CountFunc) {
	return p.PacketConn, []network.CountFunc{
		p.writeFunc,
	}
}

func (p *PacketConnCounter) Upstream() any {
	return p.PacketConn
}
