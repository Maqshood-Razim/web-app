package main

import (
	"log"
	"net/http"
)

func main() {
	
	client()
	
}

func client(){

	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	log.Println("Serving static files on port 5555")
	log.Fatal(http.ListenAndServe(":5555", nil))
}