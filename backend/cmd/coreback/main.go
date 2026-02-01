package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.ListenAndServe(":"+os.Getenv("PORT"),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "hello, world: %v", r.RemoteAddr)
		}))
}
