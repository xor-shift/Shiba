package irc

import (
	"github.com/daswf852/Shiba/common/ratelimit"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientConfig struct {
	Address string
	TLS     bool

	Nick     string
	User     string
	RealName string

	Pass string

	PingFrequency int
	PingTimeout   int
}

type ServerInformation struct {
	Capabilities map[string]string
	Modes        map[string]*ModeStore
}

func NewServerInformation() ServerInformation {
	return ServerInformation{
		Capabilities: make(map[string]string),
		Modes:        make(map[string]*ModeStore),
	}
}

type ClientInformation struct {
	Nick          string
	User          string
	DisplayedHost string
}

func NewClientInformation() ClientInformation {
	return ClientInformation{
		Nick:          "",
		User:          "",
		DisplayedHost: "",
	}
}

type Client struct {
	config ClientConfig

	workersWG *sync.WaitGroup

	outgoingQueue chan Message

	pingTicker       *time.Ticker
	pingTimeoutTimer *time.Timer

	rateLimiter ratelimit.Bucket
	connection  *Connection
	parser      *Parser

	clientInfo ClientInformation
	serverInfo ServerInformation

	callbacksMutex *sync.RWMutex
	passthroughCB  func(message Message)
	postInitCB     func()
}

func NewClient(conf ClientConfig) (*Client, error) {
	conn := NewConnection(conf.TLS, conf.Address)

	if conf.PingFrequency == 0 {
		conf.PingFrequency = 60
	}

	if conf.PingTimeout == 0 {
		conf.PingTimeout = 120
	}

	client := &Client{
		config: conf,

		workersWG: &sync.WaitGroup{},

		outgoingQueue:    make(chan Message, 32),
		pingTicker:       time.NewTicker(time.Second * (time.Duration)(conf.PingFrequency)),
		pingTimeoutTimer: time.NewTimer(time.Second * (time.Duration)(conf.PingTimeout)),

		rateLimiter: ratelimit.NewBucket(16, 500),
		connection:  conn,
		parser:      NewParser(),

		serverInfo: NewServerInformation(),
		clientInfo: NewClientInformation(),

		callbacksMutex: &sync.RWMutex{},
		passthroughCB:  func(message Message) {},
		postInitCB:     func() {},
	}

	client.clientInfo.Nick = conf.Nick
	client.clientInfo.User = conf.User

	client.connection.SetIncomingCallback(client.parserHandler)

	return client, nil
}

func (client *Client) Init() error {
	if err := client.connection.Init(); err != nil {
		return err
	}

	time.NewTimer(time.Second * 60)
	time.NewTicker(time.Second * 60)

	initialMessages := []Message{
		Message{
			Source:   "",
			Command:  "CAP",
			Params:   []string{"LS", "302"},
			Trailing: "",
		},
		Message{
			Source:   "",
			Command:  "NICK",
			Params:   []string{client.config.Nick},
			Trailing: "",
		},
		Message{
			Source:   "",
			Command:  "USER",
			Params:   []string{client.config.User, "iwxz", "*", client.config.RealName},
			Trailing: "",
		},
	}

	for _, msg := range initialMessages {
		client.outgoingQueue <- msg
		client.SendMessage(msg)
	}

	client.workersWG.Add(1)
	go client.pingWorker()

	return nil
}

func (client *Client) receiver() {
	defer client.workersWG.Done()

	buffer := make([]byte, 256)

	for {
		n, err := client.connection.Read(buffer)
		if err != nil {
			log.Println("IRC message receiver got error:", err)
			break
		}

		if _, err := client.parser.Write(buffer[:n]); err != nil {
			log.Println("Faulty message received, error:", err)
		}
	}
}

func (client *Client) sender() {
	defer client.workersWG.Done()

	running := true
	for running {
		select {
		case msg, ok := <-client.outgoingQueue:
			if !ok {
				running = false
				break
			}

			if !client.rateLimiter.NewDrop() {
				log.Println("Rate limited an IRC message")
			}

			if _, err := client.connection.Write(msg.Serialize()); err != nil {
				log.Println("Error while sending IRC message:", err)
			}
		}
	}
}

func (client *Client) parserHandler(message Message) {
	switch message.Command {
	case "PING":
		client.SendMessage(Message{
			Source:   "",
			Command:  "PONG",
			Params:   []string{},
			Trailing: message.Trailing,
		})
		client.pingTimeoutTimer.Reset(time.Second * (time.Duration)(client.config.PingTimeout))

	case "CAP":
		if message.Params[0] == "*" && message.Params[1] == "LS" {
			capabilities := strings.Split(message.Trailing, " ")
			for _, v := range capabilities {
				key := ""
				val := ""
				if strings.Contains(v, "=") {
					parts := strings.Split(v, "=")
					key = parts[0]
					val = parts[1]
				} else {
					key = v
				}
				client.serverInfo.Capabilities[key] = val
			}

			client.SendMessage(Message{
				Source:   "",
				Command:  "CAP",
				Params:   []string{"END"},
				Trailing: "",
			})
		}

	case "396":
		client.clientInfo.DisplayedHost = message.Params[1]

	case "MODE":
		nick := message.Params[0]
		modeStr := message.Trailing

		_, modeStoreExists := client.serverInfo.Modes[nick]
		if !modeStoreExists {
			store := NewModeStore()
			client.serverInfo.Modes[nick] = &store
		}

		client.serverInfo.Modes[nick].ApplyModeString(modeStr)

	case "376": //end of motd
		client.callbacksMutex.RLock()
		client.postInitCB()
		client.callbacksMutex.RUnlock()
	}

	client.callbacksMutex.RLock()
	client.passthroughCB(message)
	client.callbacksMutex.RUnlock()
}

func (client *Client) pingWorker() {
	defer client.workersWG.Done()

	for running := true; running; {
		select {
		case t, ok := <-client.pingTicker.C:
			if !ok {
				running = false
				break
			}
			client.SendMessage(Message{
				Source:   "",
				Command:  "PING",
				Trailing: strconv.FormatInt(t.Unix(), 10),
			})
		case _, ok := <-client.pingTimeoutTimer.C:
			if !ok {
				running = false
				client.Close()
				break
			}
		}
	}
}

func (client *Client) SetMessageHandler(cb func(message Message)) {
	client.callbacksMutex.Lock()
	defer client.callbacksMutex.Unlock()
	client.passthroughCB = cb
}

func (client *Client) SetPostInitCallback(cb func()) {
	client.callbacksMutex.Lock()
	defer client.callbacksMutex.Unlock()
	client.postInitCB = cb
}

func (client *Client) Wait() {
	client.workersWG.Wait()
}

func (client *Client) Close() error {
	close(client.outgoingQueue)
	client.pingTicker.Stop()
	client.pingTimeoutTimer.Stop()
	return client.connection.Close()
}

func (client *Client) SendMessage(message Message) {
	client.connection.OutgoingChannel <- message
}

func (client *Client) GetNick() string {
	return client.clientInfo.Nick
}
