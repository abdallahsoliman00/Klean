# Bugs and usage issues
- [x] Make the chat name be a summary of the chat, not the message that was sent.
- [x] Add configurations where the user saves a group of temperature settings, model used, context window and system prompt, all together.
- [x] Make the configuration sticky. i.e. let the configuration previously set be the same for the next chat.
- [x] Make the artifacts side panel resizable and give it fullscreen.
- [] Make it possible to run multiple chats in parallel. If I exit a chat without explicitly stopping token straming, then keep that running in the background where I can later enter and continue.
- [] When switching models in the middle of a conversation, find out which model is the one actually doing the talking.
- [x] If an agent loops for too long, it stops. Give it a longer loop time or make it adjustable through a config.

# New features
- [x] Make the context window larger than 128k tokens.
- [x] Save the settings to the database as well.
- [x] Make an executable that can be distributed.
- [x] Add tool calls to create artifacts and code snippets. Ideas:
    - Code snippets in the sidebar
    - Artifacts
    - Ask questions
    - Web search
- [] Make code syntax highlighting work even in the chat box. (Where if you type ``` you get syntax highlighting.)
- [x] Slightly modify the colorscheme to make it less purple, or even better, make different dark and light themes available. (Five schemes, each light and dark, in Settings → Appearance.)
- [] Give the user the option to select from different code coloschemes as well.
- [] Favourite chats.
- [] Allow for multiple system prompts. (Save system prompts to database where all you do is load the system prompt and reuse it.)
- [] Make the application download the most recent version of the tool if one is available.
- [x] Possibly migrate the backend to Rust or Go while it's still small to make cross-compilation to other languages easier than Python.
- [] Add using images. The responses API now supports this.
- [x] Also migrate from the completions, and ensure the OpenAI SDK is used.
- [] If responses endpoint doesn't exist fallback to completions.
- [] Add file uploads
- [] Add file creation (not just markdown files)
- [x] Embed the frontend in the executable.
- [] Make a tree where you can wander off on tangents and come back.
- [] Add Skills and MCPs
- [] Use either https://github.com/eKoopmans/html2pdf.js or https://github.com/SanzarRehman/HTML2PDF for html to pdf rendering for downloading.

# Repo Admin
- [x] Change the name to something more interesting than switchboard. (Now Klean.)
- [x] Make an AGENTS.md file for each of the frontend and backend.
- [x] Group this TODO file into priorities, features and bugs.
- [x] Make a CI/CD pipeline that has the most recent version of the tool available
- [] Make the README more concise and write a nice story.
- [x] Create scripts for both the AI agents and local developers:
    - Code linting and formatting fir checks
    - Code type checking
    - Run development
