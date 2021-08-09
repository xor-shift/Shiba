package commandMod

import "strings"

func ShellTokenize(source string) []string {
	tokens := make([]string, 0)

	const (
		ModeRegular         = iota
		ModeExpectingEscape = iota
		ModeInQuote         = iota
	)

	var charEscapes map[rune]string = make(map[rune]string)
	charEscapes['n'] = "\n"
	charEscapes['t'] = "\t"

	mode := ModeRegular
	preEscapeMode := ModeRegular
	singleTickQuote := false

	builder := strings.Builder{}

	AppendCurrent := func() {
		tokens = append(tokens, builder.String())
		builder.Reset()
	}

	ProcessRegular := func(c rune) {
		switch c {
		case '\\':
			preEscapeMode = mode
			mode = ModeExpectingEscape
		case '"':
			mode = ModeInQuote
			singleTickQuote = false
		case '\'':
			mode = ModeInQuote
			singleTickQuote = true
		case ' ':
			AppendCurrent()
		default:
			builder.WriteRune(c)
		}
	}

	ProcessInQuote := func(c rune) {
		switch c {
		case '\\':
			preEscapeMode = mode
			mode = ModeExpectingEscape
		case '\'':
			fallthrough
		case '"':
			if (singleTickQuote && c == '\'') || !singleTickQuote && c == '"' {
				mode = ModeRegular
			} else {
				builder.WriteRune(c)
			}
		default:
			builder.WriteRune(c)
		}
	}

	ProcessEscape := func(c rune) {
		res, ok := charEscapes[c]

		if ok {
			builder.WriteString(res)
		} else {
			builder.WriteRune(c)
		}

		mode = preEscapeMode
	}

	for _, c := range source {
		switch mode {
		case ModeRegular:
			ProcessRegular(c)
		case ModeExpectingEscape:
			ProcessEscape(c)
		case ModeInQuote:
			ProcessInQuote(c)
		}
	}

	AppendCurrent()

	return tokens
}
