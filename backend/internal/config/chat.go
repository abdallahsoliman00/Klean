package config

import "time"

// Defaults for new chats. They are also the column defaults in the SQLite schema.
const (
	DefaultContextWindow = 1_048_576 // tokens
	DefaultTemperature   = 0.7
	DefaultChatTitle     = "New chat"

	// AutoTitleChars is how much of the first message becomes the chat title.
	AutoTitleChars = 42
)

// Fitting a chat into the context window. Token counts are estimates.
const (
	ReplyShare         = 0.25 // part of the window kept free for the model's answer
	PerMessageOverhead = 4    // tokens each message costs beyond its text
	CharsPerToken      = 4    // rough, model-agnostic
)

// The database.
const (
	DBFileName = "klean.db"
	// The per-user data folder: lower case on Linux and others, capitalised
	// on macOS and Windows where folder names usually are.
	DataDirName        = "klean"
	DataDirNameDesktop = "Klean"

	DBBusyTimeout    = 10 * time.Second // how long a write waits for a lock
	DeletedRetention = 24 * time.Hour   // how long a deleted chat can be restored
)
