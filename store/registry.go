package store

import "github.com/myntra/goscheduler/constants"

type Factory func() Callback

var Registry = map[string]Factory{}

func InitializeCallbackRegistry(clientCallbacks map[string]Factory) {
	// default implementations
	defaultCallbacks := map[string]Factory{
		constants.DefaultCallback: func() Callback { return &HttpCallback{} },
	}

	for callbackType, factory := range defaultCallbacks {
		if clientFactory, ok := clientCallbacks[callbackType]; ok {
			// if the client provided an implementation, use it
			Registry[callbackType] = clientFactory
		} else {
			// otherwise use the default
			Registry[callbackType] = factory
		}
	}
}
