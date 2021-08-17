package reactionMod

import (
	"github.com/daswf852/Shiba/bot/mbus"
	"github.com/daswf852/Shiba/bot/message"
	"github.com/jmoiron/sqlx"
	"log"
	"math/rand"
	"regexp"
)

type ReactionModule struct {
	bus *mbus.Bus

	db            *sqlx.DB
	reactionStore map[string]map[string][]string
	regexCache    map[string]*regexp.Regexp
}

func New(db *sqlx.DB) *ReactionModule {
	mod := &ReactionModule{
		bus:           nil,
		db:            db,
		reactionStore: make(map[string]map[string][]string),
		regexCache:    make(map[string]*regexp.Regexp),
	}

	type DBReaction struct {
		ReplyTarget string `db:"when_replying_to"`
		RegexStr    string `db:"regex_str"`
		ReplyStr    string `db:"reply_str"`
	}

	res, err := db.Queryx("select when_replying_to, regex_str, reply_str from reactions;")
	if err != nil {
		log.Fatalln(err)
	}

	for res.Next() {
		reac := DBReaction{}
		if err := res.StructScan(&reac); err != nil {
			log.Fatalln(err)
		}

		if _, targetExists := mod.reactionStore[reac.ReplyTarget]; !targetExists {
			mod.reactionStore[reac.ReplyTarget] = make(map[string][]string)
		}

		regex, err := regexp.Compile(reac.RegexStr)
		if err != nil {
			log.Println(err)
			continue
		}

		if arr, ok := mod.reactionStore[reac.ReplyTarget][reac.RegexStr]; ok {
			mod.reactionStore[reac.ReplyTarget][reac.RegexStr] = append(arr, reac.ReplyStr)
		} else {
			mod.reactionStore[reac.ReplyTarget][reac.RegexStr] = []string{reac.ReplyStr}
		}
		mod.regexCache[reac.RegexStr] = regex
	}

	return mod
}

func (mod *ReactionModule) GetIdentifier() mbus.ModuleIdentifier {
	return mbus.ModuleIdentifier{
		MainIdent: "Module",
		SubIdent:  "Reaction",
	}
}

func (mod *ReactionModule) OnRegister(bus *mbus.Bus) {
	mod.bus = bus
	log.Println("Reaction module registered")
}

func (mod *ReactionModule) OnUnregister() {
	log.Println("Reaction module unregistered")
}

func (mod *ReactionModule) OnMessage(msg mbus.Message) {
	if incomingChatMessage, ok := msg.(mbus.IncomingChatMessage); ok {
		text := message.MessageToPlaintext(incomingChatMessage.Message)
		replyIdent := incomingChatMessage.SourceModule.String() + ":" + incomingChatMessage.ReplyTo
		if target, targetExists := mod.reactionStore[replyIdent]; targetExists {
			for regexStr, replyStrArr := range target {
				if mod.regexCache[regexStr].MatchString(text) {
					reply, err := message.FromIntermediate(replyStrArr[rand.Int()%len(replyStrArr)])
					if err != nil {
						log.Printf("Database has faulty reply_str in reactions for regex '%s', got error: %s",
							regexStr, err)
					}

					mod.bus.NewMessage(mbus.OutgoingChatMessage{
						TargetModule: incomingChatMessage.SourceModule,
						To:           incomingChatMessage.ReplyTo,
						Message:      reply,
					})
					return
				}
			}
		}
	} else if controlMessage, ok := msg.(mbus.ModuleControlMessage); ok {
		if controlMessage.StrArgv[0] == "add" {
			whenReplyingTo := controlMessage.StrArgv[1]
			regexStr := controlMessage.StrArgv[2]
			replyStr := controlMessage.StrArgv[3]
			addedBy := controlMessage.StrArgv[4]
			mod.addReaction(whenReplyingTo, regexStr, replyStr, addedBy)
		}
	}
}

func (mod *ReactionModule) addReaction(replyingTo, regexStr, replyStr, addedBy string) bool {
	if _, cached := mod.regexCache[regexStr]; !cached {
		regex, err := regexp.Compile(regexStr)
		if err != nil {
			return false
		}
		mod.regexCache[regexStr] = regex
	}

	if _, ok := mod.reactionStore[replyingTo]; !ok {
		mod.reactionStore[replyingTo] = make(map[string][]string)
	}

	if reacs, ok := mod.reactionStore[replyingTo][regexStr]; ok {
		for _, v := range reacs {
			if v == replyStr {
				return false
			}
		}
	} else {
		mod.reactionStore[replyingTo][regexStr] = make([]string, 0)
	}

	mod.reactionStore[replyingTo][regexStr] = append(mod.reactionStore[replyingTo][regexStr], replyStr)

	mod.db.MustExec("insert into reactions (when_replying_to, regex_str, reply_str, added_by) values (?, ?, ?, ?);", replyingTo, regexStr, replyStr, addedBy)

	return true
}

func (mod *ReactionModule) delReaction(replyingTo, regexStr string) bool {
	if store, ok := mod.reactionStore[replyingTo]; ok {
		if _, ok := store[regexStr]; ok {
			delete(mod.reactionStore[replyingTo], regexStr)
			_, _ = mod.db.Exec("delete from reactions where when_replying_to = ? and regex_str = ?", replyingTo, regexStr)
		} else {
			return false
		}
	} else {
		return false
	}

	return true
}
