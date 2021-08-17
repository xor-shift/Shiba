package commandMod

import (
	"github.com/daswf852/Shiba/bot/mbus"
	"github.com/daswf852/Shiba/bot/message"
	"github.com/jmoiron/sqlx"
	"log"
	"strings"
)

type UserInformation struct {
	PermLevel int
}

type CommandModule struct {
	Prefix string

	db  *sqlx.DB
	bus *mbus.Bus

	users    map[string]*UserInformation
	commands map[string]Command
}

func New(db *sqlx.DB, prefix string) *CommandModule {
	mod := &CommandModule{
		Prefix: prefix,

		db:  db,
		bus: nil,

		users:    make(map[string]*UserInformation),
		commands: make(map[string]Command),
	}

	type DBUser struct {
		Ident     string `db:"identifier"`
		PermLevel int    `db:"perm_level"`
	}

	res, err := db.Queryx("select identifier, perm_level from users;")
	if err != nil {
		log.Fatalln(err)
	}

	for res.Next() {
		user := DBUser{}
		if err := res.StructScan(&user); err != nil {
			log.Fatalln(err)
		}

		mod.users[user.Ident] = &UserInformation{
			PermLevel: user.PermLevel,
		}
	}

	return mod
}

func (mod *CommandModule) makeUserIfMissing(userIdent string) {
	if _, ok := mod.users[userIdent]; !ok {
		mod.users[userIdent] = &UserInformation{
			PermLevel: 0,
		}
		mod.bumpDB(userIdent)
	}
}

func (mod *CommandModule) bumpDB(userIdent string) {
	user := mod.users[userIdent]

	count := make([]int, 0)
	if err := mod.db.Select(&count, "select count(*) from users where identifier = ?;", userIdent); err != nil {
		log.Printf("error while bumping DB for %s (select): %s", userIdent, err)
		return
	}

	if count[0] == 0 {
		if _, err := mod.db.Exec("insert into users (identifier, perm_level) values (?, ?)",
			userIdent,
			user.PermLevel,
		); err != nil {
			log.Printf("error while bumping DB for %s (insert): %s", userIdent, err)
		}
	} else {
		if _, err := mod.db.Exec("update users set perm_level = ? where identifier = ?",
			user.PermLevel,
			userIdent,
		); err != nil {
			log.Printf("error while bumping DB for %s (update): %s", userIdent, err)
		}
	}
}

func (mod *CommandModule) GetUserPerm(userIdent string) int {
	if user, ok := mod.users[userIdent]; ok {
		return user.PermLevel
	} else {
		return 0
	}
}

func (mod *CommandModule) SetUserPerm(userIdent string, permLevel int) {
	mod.makeUserIfMissing(userIdent)
	if mod.users[userIdent].PermLevel != permLevel {
		mod.users[userIdent].PermLevel = permLevel
		mod.bumpDB(userIdent)
	}
}

func (mod *CommandModule) RegisterCommand(command Command) {
	mod.commands[command.Ident] = command
}

func (mod *CommandModule) GetIdentifier() mbus.ModuleIdentifier {
	return mbus.ModuleIdentifier{
		MainIdent: "Module",
		SubIdent:  "Command",
	}
}

func (mod *CommandModule) OnRegister(bus *mbus.Bus) {
	mod.bus = bus
	log.Println("Ping module registered")
}

func (mod *CommandModule) OnUnregister() {
	log.Println("Ping module unregistered")
}

func (mod *CommandModule) OnMessage(msg mbus.Message) {
	if inChatMessage, ok := msg.(mbus.IncomingChatMessage); ok {
		text := message.MessageToPlaintext(inChatMessage.Message)
		if !strings.HasPrefix(text, mod.Prefix) {
			return
		}

		tokens := ShellTokenize(text)
		tokens[0] = strings.TrimPrefix(tokens[0], mod.Prefix)
		if command, ok := mod.commands[tokens[0]]; !ok {
			mod.bus.NewMessage(inChatMessage.MakeReply(message.PlaintextToMessage("Invalid command")))
			return
		} else {
			if mod.GetUserPerm(inChatMessage.SenderIdent) < command.MinPerm {
				mod.bus.NewMessage(inChatMessage.MakeReply(message.PlaintextToMessage("Insufficient permission")))
				return
			}

			if command.MinArgs != -1 && command.MinArgs > len(tokens) {
				mod.bus.NewMessage(inChatMessage.MakeReply(message.PlaintextToMessage("Insufficient argument count")))
				return
			} else if command.MaxArgs != -1 && len(tokens) > command.MaxArgs {
				mod.bus.NewMessage(inChatMessage.MakeReply(message.PlaintextToMessage("Excess arguments")))
				return
			}

			command.Callback(tokens, inChatMessage, mod.bus)
		}
	} else if controlMessage, ok := msg.(mbus.ModuleControlMessage); ok {
		if controlMessage.StrArgv[0] == "setperm" {
			mod.SetUserPerm(controlMessage.StrArgv[1], controlMessage.OtherData["level"].(int))
		}
	}
}
