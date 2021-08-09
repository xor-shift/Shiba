package mbus

import "strings"

type ModuleIdentifier struct {
	MainIdent string
	SubIdent  string
}

func (mi ModuleIdentifier) Compare(other ModuleIdentifier) int {
	ret := 0
	if mi.MainIdent == other.MainIdent {
		ret++
		if other.SubIdent == "*" || mi.SubIdent == other.SubIdent {
			ret++
		}
	}
	return ret
}

func (mi ModuleIdentifier) String() string {
	return mi.MainIdent + ":" + mi.SubIdent
}

func ModuleIdentifierFromString(str string) ModuleIdentifier {
	idx := strings.Index(str, ":")
	return ModuleIdentifier{
		MainIdent: str[:idx],
		SubIdent:  str[idx+1:],
	}
}
