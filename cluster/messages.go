package cluster

// EntityIDs contains a slice of entity IDs.
type EntityIDs struct {
	Ids []string
}

// AppNames contains a slice of application names.
type AppNames struct {
	Names []string
}

// Request represents a request to be sent to a remote node.
type Request struct {
	entity   interface{} // The entity to be sent.
	method   string      // The method to be called.
	destNode string      // The destination node address.
}

// Response represents a response received from a remote node.
type Response struct {
	ServerAddress string // The server address that sent the response.
	Error         string // The error message, if any.
	Status        int    // The status of the response.
}

const (
	SUCCESS = iota // The request was successful.
	FAILED         // The request failed.
)

