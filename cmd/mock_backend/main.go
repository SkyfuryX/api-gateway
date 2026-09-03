package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type response struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

func writeJSONResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling JSON: %v\n", err)
		http.Error(w, "Internal Server Error\n", http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func mockResponder1(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request receieved from %v\n", r.RemoteAddr)
	resp := response{
		Service: "A",
		Status:  "OK",
	}
	writeJSONResponse(w, resp)
}

func mockResponder2(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request receieved from %v\n", r.RemoteAddr)
	resp := response{
		Service: "B",
		Status:  "OK",
	}
	writeJSONResponse(w, resp)
}

func main() {
	const port = "8081"
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	mux.HandleFunc("GET /1", mockResponder1)
	mux.HandleFunc("GET /2", mockResponder2)

	log.Printf("Serving on port %v\n", port)
	log.Fatal(srv.ListenAndServe())
}
