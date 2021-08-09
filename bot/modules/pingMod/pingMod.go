package pingMod

import (
	"github.com/daswf852/counting/bot/mbus"
	"github.com/daswf852/counting/bot/message"
	"log"
)

type PingModule struct {
	bus *mbus.Bus
}

func (mod *PingModule) GetIdentifier() mbus.ModuleIdentifier {
	return mbus.ModuleIdentifier{
		MainIdent: "Module",
		SubIdent:  "Ping",
	}
}

func (plat *PingModule) OnRegister(bus *mbus.Bus) {
	plat.bus = bus
	log.Println("Ping module registered")
}

func (plat *PingModule) OnUnregister() {
	log.Println("Ping module unregistered")
}

func (plat *PingModule) OnMessage(msg mbus.Message) {
	if msg, ok := msg.(mbus.IncomingChatMessage); ok {
		text := message.MessageToPlaintext(msg.Message)
		if text == "Ping" || text == "ping" {
			plat.bus.NewMessage(mbus.OutgoingChatMessage{
				TargetModule: msg.SourceModule,
				To:           msg.ReplyTo,
				Message:      message.PlaintextToMessage("Pong"),
			})
		}
	}
}
