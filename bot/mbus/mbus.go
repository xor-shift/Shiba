package mbus

import (
	"log"
	"sync"
)

type Bus struct {
	workersWG *sync.WaitGroup
	//all actions should read lock this, module modifications etc. will write lock it
	busMutex *sync.RWMutex
	//no mutex for this is needed, busMutex should suffice
	modules      map[ModuleIdentifier]Module
	messageQueue chan Message
}

func New() *Bus {
	bus := &Bus{
		busMutex:     &sync.RWMutex{},
		modules:      make(map[ModuleIdentifier]Module),
		workersWG:    &sync.WaitGroup{},
		messageQueue: make(chan Message, 64),
	}

	return bus
}

func (bus *Bus) Stop() {
	bus.busMutex.Lock()
	defer bus.busMutex.Unlock()

	for _, v := range bus.modules {
		v.OnUnregister()
	}

	close(bus.messageQueue)

	bus.Wait()
}

func (bus *Bus) RunSync() {
	bus.messageWorker(false)
}

func (bus *Bus) RunAsync() {
	bus.workersWG.Add(1)
	go bus.messageWorker(true)
}

func (bus *Bus) Wait() {
	bus.workersWG.Wait()
}

func (bus *Bus) messageWorker(async bool) {
	log.Println("Message worker has started")

	if async {
		defer bus.workersWG.Done()
	}

	TriggerMessage := func(module Module, message Message) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Bus message handler for module %s panicked, More informtion:\n%T", module.GetIdentifier().String(), r)
			}
		}()

		module.OnMessage(message)
	}

	running := true
	for running {
		select {
		case msg, ok := <-bus.messageQueue:
			if !ok {
				running = false
				break
			}

			bus.busMutex.RLock()

			if tMsg, ok := msg.(TargetedMessage); ok {
				tIdent := tMsg.GetTargetIdentifier()
				if tIdent.SubIdent == "*" {
					for k, v := range bus.modules {
						if k.MainIdent == tIdent.MainIdent {
							TriggerMessage(v, msg)
						}
					}
				} else {
					if target, ok := bus.modules[tIdent]; ok {
						TriggerMessage(target, msg)
					}
				}
			} else {
				for _, v := range bus.modules {
					TriggerMessage(v, msg)
				}
			}

			bus.busMutex.RUnlock()
		}
	}

	log.Println("Message worker is exiting")
}

func (bus *Bus) RegisterModule(module Module) {
	identifier := module.GetIdentifier()

	bus.busMutex.Lock()
	defer bus.busMutex.Unlock()

	bus.unregisterModule(identifier)
	bus.modules[identifier] = module
	module.OnRegister(bus)
}

func (bus *Bus) UnregisterModule(identifier ModuleIdentifier) {
	bus.busMutex.Lock()
	defer bus.busMutex.Unlock()

	bus.UnregisterModule(identifier)
}

func (bus *Bus) unregisterModule(identifier ModuleIdentifier) {
	if mod, ok := bus.modules[identifier]; ok {
		mod.OnUnregister()
		delete(bus.modules, identifier)
	}
}

func (bus *Bus) NewMessage(message Message) {
	bus.messageQueue <- message
}
