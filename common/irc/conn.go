package irc

import (
	"crypto/tls"
	"net"
	"sync"
)

type Connection struct {
	address           string
	isTLS             bool
	tlsConnection     *tls.Conn
	regularConnection net.Conn

	parser        *Parser
	connWorkersWG *sync.WaitGroup

	IncomingChannel chan Message
	OutgoingChannel chan Message
}

func NewConnection(tls bool, address string) (*Connection, error) {
	conn := &Connection{
		address:           address,
		isTLS:             tls,
		tlsConnection:     nil,
		regularConnection: nil,
	}

	if err := conn.Init(); err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Connection) Init() error {
	if c.isTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS10,
		}

		conn, connErr := tls.Dial("tcp", c.address, tlsConfig)
		if connErr != nil {
			return connErr
		}

		c.tlsConnection = conn
	} else {
		conn, connErr := net.Dial("tcp", c.address)
		if connErr != nil {
			return connErr
		}

		c.regularConnection = conn
	}

	return nil
}

func (c *Connection) Close() error {
	if c.isTLS {
		return c.tlsConnection.Close()
	} else {
		return c.regularConnection.Close()
	}
}

func (c *Connection) Write(b []byte) (int, error) {
	if c.isTLS {
		return c.tlsConnection.Write(b)
	} else {
		return c.regularConnection.Write(b)
	}
}

func (c *Connection) Read(b []byte) (int, error) {
	if c.isTLS {
		return c.tlsConnection.Read(b)
	} else {
		return c.regularConnection.Read(b)
	}
}
