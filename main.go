package main

import (
	"log"
	"net/http"

	"github.com/solidredcompany/spotify-queue-shuffle/internal/auth"
	"github.com/solidredcompany/spotify-queue-shuffle/internal/shuffle"
)

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets/"))))

	http.HandleFunc("/", shuffle.HandleHome)
	http.HandleFunc("/login", auth.HandleLogin)
	http.HandleFunc("/authenticate", auth.HandleAuthenticate)
	http.HandleFunc("/callback", auth.HandleRedirect)
	http.HandleFunc("/shuffle", shuffle.HandleShuffle)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
