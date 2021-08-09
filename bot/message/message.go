package message

const (
	EMPropBold          = 0x1
	EMPropItalic        = 0x2
	EMPropUnderline     = 0x4
	EMPropStrikeThrough = 0x8
	EMPropMonospace     = 0x10
	EMPropSpoiler       = 0x20
	EMPropAll           = EMPropBold | EMPropItalic | EMPropUnderline | EMPropStrikeThrough | EMPropMonospace | EMPropSpoiler
)

var (
	DefaultProperties = Properties{
		EnableList:  0,
		InheritList: EMPropAll,
	}

	ResetProperties = Properties{
		EnableList:  0,
		InheritList: 0,
	}
)

type PropertyList uint32

type Properties struct {
	EnableList  PropertyList
	InheritList PropertyList
}

type MessageNode struct {
	Props Properties
	Text  string
}

type Message []MessageNode

func (msg Message) Walk(callback func(text string, currentProps, lastProps PropertyList)) {
	currentProperties := PropertyList(0)

	for _, node := range msg {
		oldProperties := currentProperties

		applyMask := ^node.Props.InheritList
		setMask := applyMask & node.Props.EnableList
		unsetMask := applyMask & currentProperties
		currentProperties ^= unsetMask
		currentProperties |= setMask

		callback(node.Text, currentProperties, oldProperties)
	}
}

func PlaintextToMessage(text string) Message {
	return Message{MessageNode{
		Props: ResetProperties,
		Text:  text,
	}}
}

func MessageToPlaintext(msg Message) string {
	s := ""

	msg.Walk(func(text string, currentProps, lastProps PropertyList) {
		s += text
	})

	return s
}

func (msg Message) String() string {
	return MessageToPlaintext(msg)
}

func (msg Message) Len() int {
	i := 0
	msg.Walk(func(text string, currentProps, lastProps PropertyList) {
		i += len(text)
	})
	return i
}

func (msg Message) TrimLeft(amount int) Message {
	if amount < 0 {
		return msg
	} else if amount >= msg.Len() {
		return PlaintextToMessage("")
	}

	//newMsg := make(Message, 0)

	//for _, v := range msg {}

	return msg
}
