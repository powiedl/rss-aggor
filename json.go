package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Printf("Responding with %v error: '%v'\n",code,msg);
	}
	type errResponse struct {
		Error string `json:"error"`  // { "error": "something went wrong" } - wenn man die Funktion mit der msg "something went wrong" aufgerufen hat
	}

	respondWithJSON(w, code, errResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON repsonse: %v\n",payload)
		w.WriteHeader(500)
		return
	}
	w.Header().Add("Content-Type","application/json")
	w.WriteHeader(code)
	w.Write(dat)
}