package slinkserv

// User maps a connection to a list of accounts
type User struct {
	Account *Account // List of authenticated accounts
	Client  *Client  // Client connection
	GameID  uint32   // Currently connected game ID
	SnakeID uint32   // Current ID of users snake
}

// Account is a container for user storage and has a password for auth.
type Account struct {
	ID       uint32
	Name     string
	Password string
}
