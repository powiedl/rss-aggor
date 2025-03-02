package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Hello world")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	portString := os.Getenv("PORT")
	if portString =="" {
		log.Fatal("PORT is not found in the environment")
	}
	router := chi.NewRouter()
	
	// CORS Optionen für Development setzen
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*","http://*"},
		AllowedMethods:   []string{"GET","POST","PUT","DELETE","OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	srv := &http.Server{
		Handler: router,
		Addr: ":"+portString,
	}

	log.Printf("Server starting on port %v\n",portString)
	err = srv.ListenAndServe() // hier bleibt der Code "stehen" - erst wenn ein Fehler auftritt, macht der Code weiter
	if err != nil {
		log.Fatal(err)
	}

}