package irc

import "strings"

type Message struct {
	Tags     map[string]string
	Source   string
	Command  string
	Params   []string
	Trailing string
}

func (msg Message) Serialize() []byte {
	buf := make([]byte, 0)

	if msg.Tags != nil {
		if tagLen := len(msg.Tags); tagLen != 0 {
			buf = append(buf, '@')
			i := 0
			for k, v := range msg.Tags {
				i++
				buf = append(buf, []byte(k)...)
				if len(v) != 0 {
					buf = append(buf, '=')
					buf = append(buf, []byte(v)...)
				}
				if i != tagLen {
					buf = append(buf, ';')
				}
			}
			buf = append(buf, ' ')
		}
	}

	if len(msg.Source) != 0 {
		buf = append(buf, ':')
		buf = append(buf, []byte(msg.Source)...)
		buf = append(buf, ' ')
	}

	buf = append(buf, []byte(msg.Command)...)
	buf = append(buf, ' ')

	if msg.Params != nil {
		if len(msg.Params) > 0 {
			for _, v := range msg.Params {
				buf = append(buf, []byte(v)...)
				buf = append(buf, ' ')
			}
		}
	}

	if len(msg.Trailing) > 0 {
		buf = append(buf, ':')
		buf = append(buf, []byte(msg.Trailing)...)
	}

	buf = append(buf, separator...)

	return buf
}

func ParseSource(source string) []string {
	nickSepIdx := strings.Index(source, "!")
	hostSepIdx := strings.Index(source, "@")

	if nickSepIdx == -1 || hostSepIdx == -1 || nickSepIdx > hostSepIdx {
		return []string{source}
	}

	return []string{source[:nickSepIdx], source[nickSepIdx+1 : hostSepIdx], source[hostSepIdx+1:]}
}
