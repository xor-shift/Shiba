package ircp

import (
	"github.com/xor-shift/Shiba/bot/mbus"
	"github.com/xor-shift/Shiba/bot/message"
	"github.com/xor-shift/Shiba/common/irc"
	"log"
)

type Platform struct {
	SubIdent string
	Client   *irc.Client
}

func New(subIdent string, conf irc.ClientConfig) (*Platform, error) {
	client, err := irc.NewClient(conf)
	if err != nil {
		return nil, err
	}

	return &Platform{
		SubIdent: subIdent,
		Client:   client,
	}, nil
}

func (plat *Platform) GetIdentifier() mbus.ModuleIdentifier {
	return mbus.ModuleIdentifier{
		MainIdent: "IRC",
		SubIdent:  plat.SubIdent,
	}
}

func parseIRCMessage(str string) message.Message {
	msg := make(message.Message, 0)

	currentFormat := message.PropertyList(0)
	currentText := make([]rune, 0)

	formats := map[rune]message.PropertyList{
		0x02: message.EMPropBold,
		0x1D: message.EMPropItalic,
		0x1F: message.EMPropUnderline,
		0x1E: message.EMPropStrikeThrough,
		0x11: message.EMPropMonospace,
	}

	TryAppend := func() bool {
		if len(currentText) > 0 {
			msg = append(msg, message.MessageNode{
				Props: message.Properties{
					EnableList:  currentFormat,
					InheritList: 0,
				},
				Text: string(currentText),
			})
			return true
		}
		return false
	}

	for _, r := range str {
		if f, ok := formats[r]; ok {
			if TryAppend() {
				currentText = make([]rune, 0)
			}
			currentFormat ^= f
		} else {
			currentText = append(currentText, r)
		}
	}

	TryAppend()

	return msg
}

func (plat *Platform) OnRegister(bus *mbus.Bus) {
	plat.Client.SetMessageHandler(func(msg irc.Message) {
		if msg.Command == "PRIVMSG" {
			replyTarget := msg.Params[0]
			if replyTarget == plat.Client.GetNick() {
				replyTarget = irc.ParseSource(msg.Source)[0]
			}

			bus.NewMessage(mbus.IncomingChatMessage{
				SourceModule: plat.GetIdentifier(),
				SenderIdent:  plat.GetIdentifier().String() + ":" + msg.Source,
				ReplyTo:      replyTarget,
				Message:      parseIRCMessage(msg.Trailing),
			})
		}
	})

	log.Println("IRC platform registered")
}

func (plat *Platform) OnUnregister() {
	_ = plat.Client.Close()
	plat.Client.Wait()
	log.Println("IRC platform unregistered")
}

func (plat *Platform) OnMessage(msg mbus.Message) {
	if outChatMSG, ok := msg.(mbus.OutgoingChatMessage); ok {
		plat.Client.SendMessage(irc.Message{
			Command:  "PRIVMSG",
			Params:   []string{outChatMSG.To},
			Trailing: message.MessageToPlaintext(outChatMSG.Message),
		})
	} else if controlMSG, ok := msg.(mbus.ModuleControlMessage); ok {
		switch controlMSG.StrArgv[0] {
		case "join":
			plat.join(controlMSG.StrArgv[1])
		}
	}
}

func (plat *Platform) join(ch string) {
	plat.Client.SendMessage(irc.Message{
		Command: "JOIN",
		Params:  []string{ch},
	})
}
