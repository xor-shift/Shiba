package main

import (
	"fmt"
	"github.com/daswf852/Shiba/bot/mbus"
	"github.com/daswf852/Shiba/bot/message"
	"github.com/daswf852/Shiba/bot/modules/commandMod"
	"github.com/daswf852/Shiba/bot/modules/reactionMod"
	ircPlat "github.com/daswf852/Shiba/bot/platforms/ircp"
	tPlat "github.com/daswf852/Shiba/bot/platforms/terminal"
	"github.com/daswf852/Shiba/common/irc"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	db  *sqlx.DB
	bus = mbus.New()
)

func prepIRC() {
	res, err := db.Queryx("select subident, auto_join, address, tls, nick_name, user_name, real_name, pass, ping_freq, ping_timeout from irc_configs;")
	if err != nil {
		log.Fatalln(err)
	}

	type DBIRCConfig struct {
		SubIdent string `db:"subident"`
		AutoJoin string `db:"auto_join"`

		Address string `db:"address"`
		TLS     bool   `db:"tls"`

		Nick     string `db:"nick_name"`
		User     string `db:"user_name"`
		RealName string `db:"real_name"`

		Pass string `db:"pass"`

		PingFrequency int `db:"ping_freq"`
		PingTimeout   int `db:"ping_timeout"`
	}

	for res.Next() {
		conf := DBIRCConfig{}
		if err := res.StructScan(&conf); err != nil {
			log.Fatalln(err)
		}

		platform, err := ircPlat.New(conf.SubIdent, irc.ClientConfig{
			Address:       conf.Address,
			TLS:           conf.TLS,
			Nick:          conf.Nick,
			User:          conf.User,
			RealName:      conf.RealName,
			Pass:          conf.Pass,
			PingFrequency: conf.PingFrequency,
			PingTimeout:   conf.PingTimeout,
		})

		if err != nil {
			panic(err)
		}

		platform.Client.SetPostInitCallback(func() {
			for _, ch := range strings.Split(conf.AutoJoin, ";") {
				bus.NewMessage(mbus.ModuleControlMessage{
					TargetModule: platform.GetIdentifier(),
					StrArgv:      []string{"join", ch},
				})
			}
		})

		if err := platform.Client.Init(); err != nil {
			panic(err)
		}

		bus.RegisterModule(platform)
	}
}

func registerCommands(module *commandMod.CommandModule) {
	module.RegisterCommand(commandMod.Command{
		Ident:   "permTest",
		Desc:    "impossiburu",
		MinPerm: 9001,
		MinArgs: -1,
		MaxArgs: -1,
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			bus.NewMessage(origMessage.MakeReply(message.PlaintextToMessage("How did you execute this command")))
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "echo",
		Desc:    "(((echo))), strips formatting before echoing, maybe",
		MinPerm: 0,
		MinArgs: 1,
		MaxArgs: -1,
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			text := origMessage.Message.String()
			idx := strings.Index(text, argv[0])
			text = text[idx+len(argv[0])+1:]
			bus.NewMessage(origMessage.MakeReply(message.PlaintextToMessage(text)))
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "mbmc",
		Desc:    "sends a (m)essage (b)us (m)odule (c)ontrol (retarded name) message to the module bus with no reply recipient. first argument is the compact module ident (IRC:AB, Module:Command, etc.)",
		MinPerm: 100,
		MinArgs: -1,
		MaxArgs: -1,
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifierFromString(argv[1]),
				StrArgv:      argv[2:],
				OtherData:    nil,
			})
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "whoami",
		Desc:    "whoami",
		MinPerm: 0,
		MinArgs: 1,
		MaxArgs: 1,
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			builder := strings.Builder{}
			builder.WriteString(fmt.Sprintf("Ident: %s", origMessage.SenderIdent))
			bus.NewMessage(origMessage.MakeReply(message.PlaintextToMessage(builder.String())))
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "setperm",
		Desc:    "",
		MinPerm: 100,
		MinArgs: 3,
		MaxArgs: 3,
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			i, err := strconv.Atoi(argv[2])
			if err != nil {
				bus.NewMessage(origMessage.MakeReply(message.PlaintextToMessage("Bad permission integer")))
				return
			}
			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifier{
					MainIdent: "Module",
					SubIdent:  "Command",
				},
				StrArgv:   []string{"setperm", argv[1]},
				OtherData: map[string]interface{}{"level": i},
			})
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "addr",
		Desc:    "",
		MinPerm: 10,
		MinArgs: 3,
		MaxArgs: 3,
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifier{"Module", "Reaction"},
				StrArgv:      []string{"add", origMessage.SourceModule.String() + ":" + origMessage.ReplyTo, argv[1], argv[2], origMessage.SenderIdent},
				OtherData:    nil,
			})
		},
	})

	/*
		module.RegisterCommand(commandMod.Command{
			Ident:   "stub",
			Desc:    "stub",
			MinPerm: 0,
			MinArgs: 1,
			MaxArgs: -1,
			Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			},
		})
	*/
}

func init() {
	var err error

	db, err = sqlx.Connect("sqlite3", os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	prepIRC()

	cmdMod := commandMod.New(db, ";")
	registerCommands(cmdMod)

	bus.RegisterModule(tPlat.New("std"))
	bus.RegisterModule(reactionMod.New(db))
	bus.RegisterModule(cmdMod)
}

func main() {
	bus.RunAsync()
	bus.Wait()
}
