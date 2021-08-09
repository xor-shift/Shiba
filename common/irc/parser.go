package irc

import (
	"bytes"
	"regexp"
	"strings"
)

const (
	messageRegexString = `(?:@([^@ ]+) +)?` + //tags
		`(?::([^ ]+) +)?` + //prefix
		`((?:[a-zA-Z]+)|(?:[0-9]{3}))` + //command
		`((?: +[^ \0:]+)+)?` + //params
		`(?: +:([^\0]+))` //trailing
)

var (
	//messageRegex = regexp.MustCompile(`(?:@([^@ ]+) +)?(?::([^ ]+) +)?((?:[a-zA-Z]+)|(?:[0-9]{3}))((?: +[^ \n\r\0:]+)+)?(?: +:([^\n\r\0]+))`)
	messageRegex = regexp.MustCompile(messageRegexString)
	separator    = []byte{'\r', '\n'}
)

type Parser struct {
	callback func(Message)
	buffer   []byte
}

func NewParser() *Parser {
	return &Parser{
		callback: func(message Message) {},
		buffer:   make([]byte, 0),
	}
}

func (p *Parser) SetCallback(cb func(Message)) {
	p.callback = cb
}

func (p *Parser) Write(buf []byte) (int, error) {
	l := len(buf)

	p.buffer = append(p.buffer, buf...)

	for i := bytes.Index(p.buffer, separator); i != -1; i = bytes.Index(p.buffer, separator) {
		current := p.buffer[:i]
		p.buffer = p.buffer[i+2:]
		res := messageRegex.FindAllSubmatch(current, -1)
		if len(res) > 0 {
			res := res[0]

			paramsStr := strings.TrimSpace(string(res[4]))
			paramsStr = regexp.MustCompile(" +").ReplaceAllString(paramsStr, " ")

			p.callback(Message{
				Tags:     map[string]string{},
				Source:   string(res[2]),
				Command:  string(res[3]),
				Params:   strings.Split(paramsStr, " "),
				Trailing: string(res[5]),
			})
		}
	}

	return l, nil
}

func (p *Parser) Close() error {
	return nil
}
