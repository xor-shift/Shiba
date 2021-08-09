package irc

type ModeStore struct {
	store map[rune]struct{}
}

func NewModeStore() ModeStore {
	return ModeStore{store: make(map[rune]struct{})}
}

func (store ModeStore) HasMode(mode rune) bool {
	_, ok := store.store[mode]
	return ok
}

func (store *ModeStore) AddMode(mode rune) {
	store.store[mode] = struct{}{}
}

func (store *ModeStore) RemoveMode(mode rune) {
	if store.HasMode(mode) {
		delete(store.store, mode)
	}
}

func (store *ModeStore) ApplyModeString(modes string) {
	additive := true
	for _, ch := range modes {
		switch ch {
		case '+':
			additive = true
			break
		case '-':
			additive = false
			break
		default:
			if additive {
				store.AddMode(ch)
			} else {
				store.RemoveMode(ch)
			}
		}
	}
}
