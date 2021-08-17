package irc

import (
	"crypto/tls"
	"github.com/daswf852/Shiba/common/ratelimit"
	"log"
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

	incomingCallback func(Message)
	IncomingChannel  chan Message
	rateLimiter      ratelimit.Bucket
	OutgoingChannel  chan Message
}

func NewConnection(tls bool, address string) *Connection {
	conn := &Connection{
		address:           address,
		isTLS:             tls,
		tlsConnection:     nil,
		regularConnection: nil,

		parser:        NewParser(),
		connWorkersWG: &sync.WaitGroup{},

		incomingCallback: nil,
		IncomingChannel:  make(chan Message, 128),
		rateLimiter:      ratelimit.NewBucket(16, 400),
		OutgoingChannel:  make(chan Message, 128),
	}

	conn.parser.SetCallback(conn.parserHandler)

	return conn
}

//SetIncomingCallback will set the callback function to invoke on new messages and will practically disable IncomingChannel. Must be called before Init()
func (conn *Connection) SetIncomingCallback(fn func(Message)) {
	conn.incomingCallback = fn
}

//Init establishes a connection and starts necessary workers etc.
func (conn *Connection) Init() error {
	if conn.isTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS10,
		}

		tlsConn, connErr := tls.Dial("tcp", conn.address, tlsConfig)
		if connErr != nil {
			return connErr
		}

		conn.tlsConnection = tlsConn
	} else {
		netConn, connErr := net.Dial("tcp", conn.address)
		if connErr != nil {
			return connErr
		}

		conn.regularConnection = netConn
	}

	conn.connWorkersWG.Add(2)
	go conn.incomingWorker()
	go conn.outgoingWorker()

	return nil
}

func (conn *Connection) Close() error {
	if conn.isTLS {
		return conn.tlsConnection.Close()
	} else {
		return conn.regularConnection.Close()
	}
}

func (conn *Connection) Write(b []byte) (int, error) {
	if conn.isTLS {
		return conn.tlsConnection.Write(b)
	} else {
		return conn.regularConnection.Write(b)
	}
}

func (conn *Connection) Read(b []byte) (int, error) {
	if conn.isTLS {
		return conn.tlsConnection.Read(b)
	} else {
		return conn.regularConnection.Read(b)
	}
}

func (conn *Connection) parserHandler(message Message) {
	if conn.incomingCallback != nil {
		conn.incomingCallback(message)
	} else {
		conn.IncomingChannel <- message
	}
}

func (conn *Connection) incomingWorker() {
	defer conn.connWorkersWG.Done()

	buffer := make([]byte, 256)

	for running := true; running; {
		n, err := conn.Read(buffer)
		if err != nil {
			running = false
			log.Println("IRC message receiver got error:", err)
			break
		}

		if _, err := conn.parser.Write(buffer[:n]); err != nil {
			log.Println("Faulty message received, error:", err)
		}
	}

	if err := conn.Close(); err != nil {
		log.Println("Got error while closing IRC connection:", err)
	}
}

func (conn *Connection) outgoingWorker() {
	defer conn.connWorkersWG.Done()

	running := true
	for running {
		select {
		case msg, ok := <-conn.OutgoingChannel:
			if !ok {
				running = false
				break
			}

			if !conn.rateLimiter.NewDrop() {
				log.Println("Rate limited an IRC message")
			}

			if _, err := conn.Write(msg.Serialize()); err != nil {
				log.Println("Error while sending IRC message:", err)
			}
		}
	}
}
