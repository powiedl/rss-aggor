package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/powiedl/rss-aggor/internal/database"
)

type apiConfig struct {
	DB *database.Queries
}

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
	dbUrl := os.Getenv("DB_URL")
	if dbUrl =="" {
		log.Fatal("DB_URL is not found in the environment")
	}
	
	conn,err := sql.Open("postgres",dbUrl)
	if err != nil {
		log.Fatal("Can't connect to database:",err)
	}

	if err != nil {
		log.Fatal("Can't create database connection:",err)
	}

	db := database.New(conn)
	apiCfg := apiConfig {
		DB: db,
	}

	go startScraping(db,2,time.Minute)

	

	router := chi.NewRouter()
	
	// CORS Optionen für Development setzen
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*","http://*"},
		AllowedMethods:   []string{"GET","POST","PUT","DELETE","OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	v1Router := chi.NewRouter()
	v1Router.Get("/healthz",handlerReadiness)
	v1Router.Get("/err",handlerErr)
	v1Router.Post("/users",apiCfg.handlerCreateUser)
	v1Router.Get("/users",apiCfg.middlewareAuth(apiCfg.handlerGetUser))
	
	v1Router.Get("/posts",apiCfg.middlewareAuth(apiCfg.handlerGetPostsForUser))

	v1Router.Post("/feeds",apiCfg.middlewareAuth(apiCfg.handlerCreateFeed))
	v1Router.Get("/feeds",apiCfg.handlerGetFeeds)

	v1Router.Post("/feed_follows",apiCfg.middlewareAuth(apiCfg.handlerCreateFeedFollow))
	v1Router.Get("/feed_follows",apiCfg.middlewareAuth(apiCfg.handlerGetFeedFollows))
	v1Router.Delete("/feed_follows/{feedFollowID}",apiCfg.middlewareAuth(apiCfg.handlerDeleteFeedFollows))
	
	router.Mount("/v1",v1Router)


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

