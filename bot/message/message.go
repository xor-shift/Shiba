package message

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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

		currentProperties &= ^node.Props.InheritList
		currentProperties |= (node.Props.InheritList & oldProperties) | node.Props.EnableList

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

//Flatten just removes the need for inheritance
func (msg Message) Flatten() Message { //TODO: pick a better name
	i := 0

	newMsg := make(Message, len(msg))

	msg.Walk(func(text string, currentProps, lastProps PropertyList) {
		newMsg[i] = MessageNode{
			Props: Properties{
				EnableList:  currentProps,
				InheritList: 0,
			},
			Text: text,
		}
		i++
	})

	return newMsg
}

func (msg Message) StrictlyEquals(other Message) bool {
	if len(msg) != len(other) {
		return false
	}

	for k, v := range msg {
		n0 := v
		n1 := other[k]

		if (n0.Text != n1.Text) ||
			(n0.Props.EnableList != n1.Props.EnableList) ||
			(n0.Props.InheritList != n1.Props.InheritList) {
			return false
		}
	}

	return true
}

func (msg Message) VisiblyEquals(other Message) bool { //TODO: fix lazy implementation
	return msg.Flatten().ToIntermediate() == other.Flatten().ToIntermediate()
}

func (msg Message) TrimLeft(amount int) Message {
	if amount >= msg.Len() {
		return make(Message, 0)
	}

	msg = msg.Flatten()

	stripUpTo := 0

	for k, v := range msg {
		if amount > len(v.Text) {
			amount -= len(v.Text)
		} else {
			stripUpTo = k
			break
		}
	}

	msg = msg[stripUpTo:]
	msg[0].Text = msg[0].Text[amount:]

	return msg
}

func (msg Message) Index(substring string) int { //TODO: holy fuck is this lazy
	return strings.Index(msg.String(), substring)
}

func (msg Message) TrimPrefix(prefix string) Message {
	idx := msg.Index(prefix)
	if idx != 0 {
		return msg
	}

	return msg.TrimLeft(idx + len(prefix))
}

//ToIntermediate translates a message to an intermediate form to later store as a string
func (msg Message) ToIntermediate() string {
	//the format is: enable:inherit:strLen:theMessage, repeated

	builder := strings.Builder{}

	for _, node := range msg {
		builder.WriteString(fmt.Sprintf("%d:%d:%d:%s",
			node.Props.EnableList,
			node.Props.InheritList,
			len(node.Text),
			node.Text,
		))
	}

	return builder.String()
}

func FromIntermediate(str string) (Message, error) {
	msg := make(Message, 0)

	ExtractOne := func() (int64, error) {
		if idx := strings.Index(str, ":"); idx == -1 {
			return -1, errors.New("couldn't find an integer segment in the string")
		} else if i, err := strconv.ParseInt(str[:idx], 10, 64); err != nil {
			return -1, errors.New("bad integer segment in the string")
		} else {
			str = str[idx+1:]
			return i, nil
		}
	}

	for len(str) > 0 {
		vals := [3]int64{-1, -1, -1}

		for i := 0; i < 3; i++ {
			if j, err := ExtractOne(); err != nil {
				return nil, err
			} else {
				vals[i] = j
			}
		}

		if vals[2] == -1 { //rest of the string, to read older entries that didn't support this thing
			vals[2] = int64(len(str))
		}

		msg = append(msg, MessageNode{
			Props: Properties{
				EnableList:  PropertyList(vals[0]),
				InheritList: PropertyList(vals[1]),
			},
			Text: str[:int(vals[2])],
		})

		str = str[int(vals[2]):]
	}

	return msg, nil
}
