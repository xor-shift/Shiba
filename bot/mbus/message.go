package mbus

import "github.com/xor-shift/Shiba/bot/message"

const (
	MTypGeneric              = iota
	MTypIncomingChat         = iota
	MTypOutgoingChat         = iota
	MTypNewModuleRegistered  = iota
	MTypModuleControlMessage = iota
)

type Message interface {
	GetType() int
}

type TargetedMessage interface {
	Message
	GetTargetIdentifier() ModuleIdentifier
}

type IncomingChatMessage struct {
	SourceModule ModuleIdentifier
	SenderIdent  string
	ReplyTo      string
	Message      message.Message
}

func (msg IncomingChatMessage) GetType() int { return MTypIncomingChat }

func (msg IncomingChatMessage) MakeReply(replyMessage message.Message) OutgoingChatMessage {
	return OutgoingChatMessage{
		TargetModule: msg.SourceModule,
		To:           msg.ReplyTo,
		Message:      replyMessage,
	}
}

type OutgoingChatMessage struct {
	TargetModule ModuleIdentifier
	To           string
	Message      message.Message
}

func (msg OutgoingChatMessage) GetType() int                          { return MTypOutgoingChat }
func (msg OutgoingChatMessage) GetTargetIdentifier() ModuleIdentifier { return msg.TargetModule }

type ModuleRegisteredMessage struct {
	TheModule ModuleIdentifier
}

func (msg ModuleRegisteredMessage) GetType() int { return MTypNewModuleRegistered }

type ModuleControlMessage struct {
	TargetModule ModuleIdentifier
	StrArgv      []string
	OtherData    map[string]interface{}
}

func (msg ModuleControlMessage) GetType() int                          { return MTypModuleControlMessage }
func (msg ModuleControlMessage) GetTargetIdentifier() ModuleIdentifier { return msg.TargetModule }
