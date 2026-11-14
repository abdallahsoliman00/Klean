// A fake OpenAI-compatible provider for UI work without a real model.
// Run: go run ./backend/cmd/fakeprovider 9100   -> base URL http://127.0.0.1:9100/v1
// Add "completions" after the port for a provider without the Responses API.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"klean/backend/internal/config"
	"klean/backend/internal/fakeprovider"
)

func main() {
	port := config.FakeProviderPort
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	fake := fakeprovider.New()
	if len(os.Args) > 2 && os.Args[2] == config.FakeProviderCompletionsArg {
		fake = fakeprovider.NewCompletionsOnly()
	}
	fmt.Printf("Fake provider at http://127.0.0.1:%s/v1\n", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, fake.Handler()))
}
