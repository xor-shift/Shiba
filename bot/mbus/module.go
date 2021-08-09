package mbus

type Module interface {
	GetIdentifier() ModuleIdentifier
	OnRegister(bus *Bus)
	OnUnregister()
	OnMessage(message Message)
}
