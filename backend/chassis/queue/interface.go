package queue

// Config - unified configuration for queue service
type Config struct {
	Name string
	URL  string

	//AWS specified
	Region             string
	CredentialsFile    string
	CredentialsProfile string
	Retries            int
}

// RecvMessage unified presentation for queue message
type RecvMessage struct {
	ID      string
	Body    string
	Handler string
}

// Client interface for queue interaction (SQS Based)
type Client interface {
	SendMessage(message string) error
	ReceiveMessage() (*RecvMessage, error)
	Acknowledge(*RecvMessage) error
}
