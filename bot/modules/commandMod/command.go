package commandMod

import "github.com/xor-shift/Shiba/bot/mbus"

type Command struct {
	Ident    string
	Desc     string
	MinPerm  int
	MinArgs  int
	MaxArgs  int
	Callback func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus)
}
