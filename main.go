package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	App     string `json:"app"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{
			Status:  "ok",
			Message: "Hello from EC2",
			App:     "myapp",
		})
	})

	// ":4444" binds 0.0.0.0 (all interfaces), NOT just localhost —
	// this is what lets the external curl reach it (avoids the DEBUG.md hang/refused trap).
	const addr = ":4444"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
