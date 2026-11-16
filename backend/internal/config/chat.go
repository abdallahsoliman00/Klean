package config

import "time"

// Defaults for new chats. They are also the column defaults in the SQLite schema.
const (
	DefaultContextWindow = 1_048_576 // tokens
	DefaultTemperature   = 0.7
	DefaultChatTitle     = "New chat"

	// ImageChatTitle is the provisional title of a chat whose first message
	// is images without text.
	ImageChatTitle = "Image"

	// AutoTitleChars is how much of the first message becomes the chat title.
	AutoTitleChars = 42
)

// Naming a chat after its first reply: the chat's model is asked for a title
// that summarises the conversation, which replaces the provisional one.
const (
	TitleTimeout    = 20 * time.Second // the whole title request; past it the provisional title stays
	TitleInputChars = 2000             // of the first message, and of the reply, shown to the model

	// TitleInstructions is the system prompt of the title request.
	TitleInstructions = "Write a title of at most six words that summarises the conversation. " +
		"Answer with the title only: no quotes, no trailing punctuation, in the language of the conversation."
)

// Intro opens every chat's instructions and tells the model where it is
// running. Its one verb is the model id the user picked. The user's own system
// prompt follows it, then the tool and skill guidance.
const Intro = "You are an AI assistant running in Klean, a desktop chat app that connects the user to language models " +
	"from providers of their choosing. In this chat the user selected the model with the id %q; other messages in " +
	"the conversation may have been written by a different model. Klean renders your replies as Markdown, with " +
	"syntax-highlighted code blocks and LaTeX math ($...$ inline, $$...$$ displayed). You can only do what your " +
	"tools allow: do not claim to have run, saved or looked up anything you did not. If the user gives their own " +
	"instructions below, they take precedence over this introduction."

// Fitting a chat into the context window. Token counts are estimates.
const (
	ReplyShare         = 0.25 // part of the window kept free for the model's answer
	PerMessageOverhead = 4    // tokens each message costs beyond its text
	CharsPerToken      = 4    // rough, model-agnostic
	// ImageTokens is what one attached image is counted as. Providers charge
	// by size and detail; this is a middling figure for a photo.
	ImageTokens = 1000
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
