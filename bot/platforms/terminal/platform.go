package terminal

import (
	"github.com/xor-shift/Shiba/bot/mbus"
	"log"
)

type Platform struct {
	SubIdent string
}

func New(subIdent string) *Platform {
	return &Platform{SubIdent: subIdent}
}

func (plat *Platform) GetIdentifier() mbus.ModuleIdentifier {
	return mbus.ModuleIdentifier{
		MainIdent: "Terminal",
		SubIdent:  plat.SubIdent,
	}
}

func (plat *Platform) OnRegister(bus *mbus.Bus) {
	log.Println("Terminal platform registered")
}

func (plat *Platform) OnUnregister() {
	log.Println("Terminal platform unregistered")
}

func (plat *Platform) OnMessage(message mbus.Message) {
	log.Println(message)
}
