package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/xor-shift/Shiba/bot/mbus"
	"github.com/xor-shift/Shiba/bot/message"
	"github.com/xor-shift/Shiba/bot/modules/commandMod"
	"github.com/xor-shift/Shiba/bot/modules/reactionMod"
	ircPlat "github.com/xor-shift/Shiba/bot/platforms/ircp"
	tPlat "github.com/xor-shift/Shiba/bot/platforms/terminal"
	"github.com/xor-shift/Shiba/common/irc"

	// _ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"gopkg.in/yaml.v2"
)

var (
	db  *sqlx.DB
	bus = mbus.New()
)

type YmlConfig struct {
	Networks []YmlNetwork `yaml:"networks"`
}

type YmlNetwork struct {
	SubIdent string   `yaml:"name"`
	Channels []string `yaml:"channels"`

	Address string `yaml:"host"`
	Port    string `yaml:"port"`
	TLS     bool   `yaml:"tls"`

	Nick     string `yaml:"nick"`
	User     string `yaml:"username"`
	RealName string `yaml:"realname"`

	Pass string `yaml:"password"`

	PingFrequency int `yaml:"ping_freq"`
	PingTimeout   int `yaml:"ping_timeout"`
}

func readConf(filename string) (*YmlConfig, error) {
	f, err := os.Open(filename)

	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg YmlConfig
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func prepIRC() {
	log.Println("Parsing IRC config...")
	networkConf, err := readConf("irc_config.yml")

	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Initialising networks...")

	for _, conf := range networkConf.Networks {
		platform, err := ircPlat.New(conf.SubIdent, irc.ClientConfig{
			Address:       conf.Address + ":" + conf.Port,
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
			for _, ch := range conf.Channels {
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
		Ident:   "gibadmin",
		Desc:    "Generates serverside secret to auth and grant admin permissions on sender",
		MinPerm: 0,
		MinArgs: 1, // blank to generate
		MaxArgs: 2, // or provide <secret> to auth
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			if len(argv) > 1 {
				// Attempt to auth with secret
				bus.NewMessage(mbus.ModuleControlMessage{
					TargetModule: mbus.ModuleIdentifier{
						MainIdent: "Module",
						SubIdent:  "Command",
					},
					StrArgv:   []string{"auth_token", argv[1]},
					OtherData: map[string]interface{}{"sender_identity": origMessage.SenderIdent},
				})
				return
			}
			// Generate a secret token to be used to grant admin for calling user ident
			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifier{
					MainIdent: "Module",
					SubIdent:  "Command",
				},
				StrArgv:   []string{"gen_token"},
				OtherData: map[string]interface{}{"sender_identity": origMessage.SenderIdent},
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
				StrArgv: []string{
					"add",
					origMessage.SourceModule.String() + ":" + origMessage.ReplyTo,
					argv[1], // regexStr
					origMessage.Message.TrimLeft(origMessage.Message.Index(argv[1]) + len(argv[1]) + 1).ToIntermediate(), // replyStr
					origMessage.SenderIdent, // addedBy
				},
				OtherData: nil,
			})
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "delr",
		Desc:    "Delete reaction by supplied regex. Or by passing in a reaction id: -id <rid>",
		MinPerm: 11,
		MinArgs: 2, // regex expression to delete
		MaxArgs: 3, // or pass in an id with delr -id <reaction id>
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			args := []string{"delete", origMessage.SourceModule.String(), origMessage.ReplyTo, origMessage.SenderIdent}
			if len(argv) > 1 {
				args = append(args, argv[1:]...)
			}

			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifier{"Module", "Reaction"},
				StrArgv:      args,
				OtherData:    nil,
			})
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "listr",
		Desc:    "List reactions by supplied regex or leave blank to list all reaction",
		MinPerm: 0,
		MinArgs: 1, // blank to list all
		MaxArgs: 2, // phrase or regex to search reactions
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			args := []string{"list", origMessage.SourceModule.String(), origMessage.ReplyTo}
			if len(argv) > 1 {
				args = append(args, argv[1:]...)
			}
			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifier{"Module", "Reaction"},
				StrArgv:      args,
				OtherData:    nil,
			})
		},
	})

	module.RegisterCommand(commandMod.Command{
		Ident:   "listrfor",
		Desc:    "List reactions that are triggered by supplied string message",
		MinPerm: 0,
		MinArgs: 2, // blank to list all
		MaxArgs: 2, // phrase search reactions
		Callback: func(argv []string, origMessage mbus.IncomingChatMessage, bus *mbus.Bus) {
			args := []string{"list_for", origMessage.SourceModule.String(), origMessage.ReplyTo}
			if len(argv) > 1 {
				args = append(args, argv[1:]...)
			}
			bus.NewMessage(mbus.ModuleControlMessage{
				TargetModule: mbus.ModuleIdentifier{"Module", "Reaction"},
				StrArgv:      args,
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

	//TODO
	_ = [1]string{
		"update reactions set reply_str='0:0:-1:' || reply_str;",
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
