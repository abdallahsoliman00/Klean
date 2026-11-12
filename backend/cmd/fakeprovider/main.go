// A fake OpenAI-compatible provider for UI work without a real model.
// Run: go run ./cmd/fakeprovider 9100   -> base URL http://127.0.0.1:9100/v1
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"klean/backend/internal/fakeprovider"
)

func main() {
	port := "9100"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	fmt.Printf("Fake provider at http://127.0.0.1:%s/v1\n", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, fakeprovider.New().Handler()))
}
