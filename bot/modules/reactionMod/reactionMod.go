package reactionMod

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/daswf852/Shiba/bot/mbus"
	"github.com/daswf852/Shiba/bot/message"
	"github.com/jmoiron/sqlx"
)

type DBReaction struct {
	Id          int64          `db:"id"`
	ReplyTarget string         `db:"when_replying_to"`
	RegexStr    string         `db:"regex_str"`
	ReplyStr    string         `db:"reply_str"`
	AddedBy     string         `db:"added_by"`
	DeletedBy   sql.NullString `db:"deleted_by"`
	CreatedAt   string         `db:"created_at"`
	UpdatedAt   string         `db:"updated_at"`
	DeletedAt   sql.NullString `db:"deleted_at"`
	Hits        string         `db:"hits"`
}
type ReactionModule struct {
	bus *mbus.Bus

	db            *sqlx.DB
	reactionStore map[string]map[string][]DBReaction
	regexCache    map[string]*regexp.Regexp
}

func init() {
	log.Println("Reaction Module Init...")
	rand.Seed(time.Now().UnixNano())
}

func New(db *sqlx.DB) *ReactionModule {
	mod := &ReactionModule{
		bus:           nil,
		db:            db,
		reactionStore: make(map[string]map[string][]DBReaction),
		regexCache:    make(map[string]*regexp.Regexp),
	}

	res, err := db.Queryx("select id, when_replying_to, regex_str, reply_str, added_by, deleted_by, created_at, updated_at, deleted_at, hits from reactions WHERE deleted_at IS NULL;")
	if err != nil {
		log.Fatalln(err)
	}
	// Reaction Store
	// 		ReplyTarget
	//			RegexStr
	//				[DBReaction, ...]
	for res.Next() {
		reac := DBReaction{}
		if err := res.StructScan(&reac); err != nil {
			log.Fatalln(err)
		}

		// Create reaction store entry if not exists
		if _, targetExists := mod.reactionStore[reac.ReplyTarget]; !targetExists {
			mod.reactionStore[reac.ReplyTarget] = make(map[string][]DBReaction)
		}

		regex, err := regexp.Compile(reac.RegexStr)
		if err != nil {
			log.Println(err)
			continue
		}
		// Append reaction to reaction store entry
		if arr, ok := mod.reactionStore[reac.ReplyTarget][reac.RegexStr]; ok {
			mod.reactionStore[reac.ReplyTarget][reac.RegexStr] = append(arr, reac)
		} else {
			mod.reactionStore[reac.ReplyTarget][reac.RegexStr] = []DBReaction{reac}
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

func (mod *ReactionModule) getReactionById(id int64) DBReaction {
	reac := DBReaction{}
	res, err := mod.db.Queryx("select id, when_replying_to, regex_str, reply_str, added_by, deleted_by, created_at, updated_at, deleted_at, hits from reactions;")
	if err != nil {
		log.Fatalln(err)
	}

	for res.Next() {
		if err := res.StructScan(&reac); err != nil {
			log.Fatalln(err)
		}
	}
	return reac
}

func (mod *ReactionModule) getAllReactions(replyIdent string) []DBReaction {
	var result []DBReaction

	if target, targetExists := mod.reactionStore[replyIdent]; targetExists {
		// Return all entries in cache
		for _, replyStrArr := range target {
			result = append(result, replyStrArr...)
		}
	}

	return result
}

func (mod *ReactionModule) getMatchesFromText(replyIdent string, text string) []DBReaction {
	var result []DBReaction

	if target, targetExists := mod.reactionStore[replyIdent]; targetExists {
		if len(text) > 0 {
			// Searching for matches in cache
			for regexStr, replyStrArr := range target {
				if mod.regexCache[regexStr].MatchString(text) {
					result = append(result, replyStrArr...)
				}
			}
		}
	}

	return result
}

func (mod *ReactionModule) getRegexEntries(replyIdent string, regexStr string) []DBReaction {
	var result []DBReaction

	if target, targetExists := mod.reactionStore[replyIdent]; targetExists {
		if len(regexStr) > 0 {
			// Just get from regexStr key
			if reactArr, regexStrExists := target[regexStr]; regexStrExists {
				return reactArr
			}
		}
	}

	return result
}

func (mod *ReactionModule) OnMessage(msg mbus.Message) {
	if incomingChatMessage, ok := msg.(mbus.IncomingChatMessage); ok {
		text := message.MessageToPlaintext(incomingChatMessage.Message)
		replyIdent := incomingChatMessage.SourceModule.String() + ":" + incomingChatMessage.ReplyTo
		matches := mod.getMatchesFromText(replyIdent, text)

		if len(matches) > 0 {
			picked := matches[rand.Int()%len(matches)]
			selectedResponse := picked.ReplyStr
			// log.Printf("debug: selected match: %s", selectedResponse)

			reply, err := message.FromIntermediate(selectedResponse)
			if err != nil {
				log.Printf("Database has faulty reply_str in reactions for regex: %s, got error: %s", picked.RegexStr, err)
				return
			}

			mod.bus.NewMessage(mbus.OutgoingChatMessage{
				TargetModule: incomingChatMessage.SourceModule,
				To:           incomingChatMessage.ReplyTo,
				Message:      reply,
			})
		}

	} else if controlMessage, ok := msg.(mbus.ModuleControlMessage); ok {
		if controlMessage.StrArgv[0] == "add" {
			// log.Println("add reaction args:")
			// log.Println(controlMessage.StrArgv)
			whenReplyingTo := controlMessage.StrArgv[1]
			regexStr := controlMessage.StrArgv[2]
			replyStr := controlMessage.StrArgv[3]
			addedBy := controlMessage.StrArgv[4]
			mod.addReaction(whenReplyingTo, regexStr, replyStr, addedBy)
			return
		}

		if controlMessage.StrArgv[0] == "delete" {
			// log.Println("delete reaction")
			// log.Printf("Args: %s", controlMessage.StrArgv)
			// 0 - delete
			// 1 - source module (network)
			// 2 - reply to channel
			// 3 - senderIdent
			// 4 - regexStr | or -id flag
			// 5 - reaction id // prev flag can be ignored

			replyIdent := controlMessage.StrArgv[1] + ":" + controlMessage.StrArgv[2]

			targetModule := mbus.ModuleIdentifierFromString(controlMessage.StrArgv[1])

			replyTo := controlMessage.StrArgv[2] // channel
			deletedBy := controlMessage.StrArgv[3]
			if len(controlMessage.StrArgv) == 5 {
				regexStr := controlMessage.StrArgv[4]
				// Delete by regex string
				ok := mod.delReactions(replyIdent, deletedBy, regexStr)
				replyMessage := message.PlaintextToMessage("Uh oh")
				if ok {
					replyMessage = message.PlaintextToMessage("Ok")
				}

				mod.bus.NewMessage(mbus.OutgoingChatMessage{
					TargetModule: targetModule,
					To:           replyTo,
					Message:      replyMessage,
				})
				return
			} else if len(controlMessage.StrArgv) == 6 {
				// Delete by reaction id
				if controlMessage.StrArgv[4] != "-id" {
					return
				}
				replyMessage := message.PlaintextToMessage("Uh oh")
				if rId, err := strconv.ParseInt(controlMessage.StrArgv[5], 10, 64); err == nil {
					ok := mod.delReactionById(replyIdent, deletedBy, rId)
					if ok {
						replyMessage = message.PlaintextToMessage("Ok")
					}
				}

				mod.bus.NewMessage(mbus.OutgoingChatMessage{
					TargetModule: targetModule,
					To:           replyTo,
					Message:      replyMessage,
				})
				return
			}
		}
		if controlMessage.StrArgv[0] == "list_for" {
			// 0 - list_for
			// 1 - source module (network)
			// 2 - reply to channel
			// 3 - triggerStr

			replyIdent := controlMessage.StrArgv[1] + ":" + controlMessage.StrArgv[2]

			targetModule := mbus.ModuleIdentifierFromString(controlMessage.StrArgv[1])

			replyTo := controlMessage.StrArgv[2]
			triggerStr := controlMessage.StrArgv[3]

			// Search for reaction of given trigger string
			results := mod.getMatchesFromText(replyIdent, triggerStr)

			if len(results) > 0 {
				for _, item := range results {

					m, err := message.FromIntermediate(item.ReplyStr)
					if err != nil {
						log.Printf("Database has faulty reply_str in reactions for regex: %s, got error: %s", item.RegexStr, err)
						continue
					}
					line := fmt.Sprintf("%d: %s %s", item.Id, item.RegexStr, message.MessageToPlaintext(m))
					// log.Println(line)
					replyMessage := message.PlaintextToMessage(line)

					mod.bus.NewMessage(mbus.OutgoingChatMessage{
						TargetModule: targetModule,
						To:           replyTo,
						Message:      replyMessage,
					})
				}
				return
			}
			mod.bus.NewMessage(mbus.OutgoingChatMessage{
				TargetModule: targetModule,
				To:           replyTo,
				Message:      message.PlaintextToMessage("No matches found."),
			})
			return
		}

		if controlMessage.StrArgv[0] == "list" {
			// log.Println("list reaction")
			// log.Printf("Args: %s", controlMessage.StrArgv)
			// 0 - list
			// 1 - source module (network)
			// 2 - reply to channel
			// 3 - regexStr (optional)
			replyIdent := controlMessage.StrArgv[1] + ":" + controlMessage.StrArgv[2]

			targetModule := mbus.ModuleIdentifierFromString(controlMessage.StrArgv[1])

			replyTo := controlMessage.StrArgv[2]
			// log.Printf("Target module: %s", controlMessage.TargetModule.String())
			// log.Printf("replyTo: %s", replyTo)

			if len(controlMessage.StrArgv) > 3 {
				// Search for reaction of given regexStr
				results := mod.getRegexEntries(replyIdent, controlMessage.StrArgv[3])
				// log.Printf("List Matches: %d", len(results))

				// TODO: refactor below
				if len(results) > 0 {
					for _, item := range results {

						m, err := message.FromIntermediate(item.ReplyStr)
						if err != nil {
							log.Printf("Database has faulty reply_str in reactions for regex: %s, got error: %s", item.RegexStr, err)
							continue
						}
						line := fmt.Sprintf("%d: %s %s", item.Id, item.RegexStr, message.MessageToPlaintext(m))
						// log.Println(line)
						replyMessage := message.PlaintextToMessage(line)

						mod.bus.NewMessage(mbus.OutgoingChatMessage{
							TargetModule: targetModule,
							To:           replyTo,
							Message:      replyMessage,
						})
					}
					return
				}
			} else {
				// List all
				// log.Println("Listing all reactions...")
				results := mod.getAllReactions(replyIdent)
				// log.Printf("List All Matches: %d", len(results))

				// TODO: refactor below
				if len(results) > 0 {
					for _, item := range results {

						m, err := message.FromIntermediate(item.ReplyStr)
						if err != nil {
							log.Printf("Database has faulty reply_str in reactions for regex: %s, got error: %s", item.RegexStr, err)
							continue
						}
						line := fmt.Sprintf("%d: %s %s", item.Id, item.RegexStr, message.MessageToPlaintext(m))
						// log.Println(line)
						replyMessage := message.PlaintextToMessage(line)

						mod.bus.NewMessage(mbus.OutgoingChatMessage{
							TargetModule: targetModule,
							To:           replyTo,
							Message:      replyMessage,
						})
					}
					return
				}
			}

			mod.bus.NewMessage(mbus.OutgoingChatMessage{
				TargetModule: targetModule,
				To:           replyTo,
				Message:      message.PlaintextToMessage("No regex matches found."),
			})
			return
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
		mod.reactionStore[replyingTo] = make(map[string][]DBReaction)
	}

	if reacs, ok := mod.reactionStore[replyingTo][regexStr]; ok {
		for _, v := range reacs {
			// Don't allow dupe reacts (for the same regex)
			if v.ReplyStr == replyStr {
				return false
			}
		}
	} else {
		mod.reactionStore[replyingTo][regexStr] = []DBReaction{}
	}

	result := mod.db.MustExec("insert into reactions (when_replying_to, regex_str, reply_str, added_by) values (?, ?, ?, ?);", replyingTo, regexStr, replyStr, addedBy)

	lastId, err := result.LastInsertId()

	if err != nil {
		log.Printf("Database insert error: %s", err)
		return false
	}

	// Retrieve by id
	// TODO: just set the struct fields properly and skip this step
	reactEntry := mod.getReactionById(lastId)

	// Add to memory cache
	mod.reactionStore[replyingTo][regexStr] = append(mod.reactionStore[replyingTo][regexStr], reactEntry)

	return true
}

func (mod *ReactionModule) delReactionById(replyingTo string, deletedBy string, rId int64) bool {
	if store, ok := mod.reactionStore[replyingTo]; ok {
		// Look through all entries to find matching id
		for regexStr, reactArr := range store {
			deleteIndex := -1
			for index, react := range reactArr {
				if react.Id == rId {
					log.Printf("Deleting reaction by id: %d", rId)
					_, err := mod.db.Exec("UPDATE reactions SET deleted_at = current_timestamp, deleted_by = ? WHERE id = ?;", deletedBy, rId)

					if err != nil {
						log.Printf("Database delete error: %s", err)
						return false
					}
					// Delete from cache
					// TODO: Maybe rebuild cache from db instead?
					deleteIndex = index
					break
				}
			}

			if deleteIndex > -1 {
				// Remove index from reactArr
				if len(mod.reactionStore[replyingTo][regexStr]) > 1 {
					mod.reactionStore[replyingTo][regexStr] = append(mod.reactionStore[replyingTo][regexStr][:deleteIndex],
						mod.reactionStore[replyingTo][regexStr][deleteIndex+1:]...)
				} else {
					// Remove regexStr entirely
					delete(mod.reactionStore[replyingTo], regexStr)
				}
				return true
			}
		}
	} else {
		return false
	}

	return false
}

// This will delete all reactions under the regexStr
func (mod *ReactionModule) delReactions(replyingTo, deletedBy string, regexStr string) bool {
	if store, ok := mod.reactionStore[replyingTo]; ok {
		if _, ok := store[regexStr]; ok {
			log.Printf("Deleting reactions for: %s ...", regexStr)
			delete(mod.reactionStore[replyingTo], regexStr)
			_, err := mod.db.Exec("UPDATE reactions SET deleted_at = current_timestamp, deleted_by = ? WHERE when_replying_to = ? and regex_str = ?", deletedBy, replyingTo, regexStr)

			if err != nil {
				log.Printf("Database delete error: %s", err)
				return false
			}

			return true
		}
	}
	return false
}
