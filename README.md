# RSS-AGGOR (RSS-Aggregator)

Geschrieben in Golang, nach dem Youtube Video https://www.youtube.com/watch?v=un6ZyFkqFKo

# Environment Variablen aus .env in Go

Damit man aus .env Environment Variablen auslesen kann (und nicht die "wirklichen" Umgebungsvariablen des Betriebssystems benutzen muss), kann man das Package `github.com/joho/godotenv` verwenden.

Hier der Codeteil, der das .env File lädt (Standardwert) und dann die Umgebungsvariable PORT ausliest. Das angegebene Package ist nur dafür verantwortlich, os auf das .env File "umzuleiten". Das eigentliche Auslesen der Umgebungsvariablen erfolgt dann mit dem Standard Go Package `os`.

```golang
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	portString := os.Getenv("PORT")
```

# HTTP Server

Wir verwenden den [go-chi Router](https://github.com/go-chi/chi) als http Router.

```sh
go get github.com/go-chi/chi
go get github.com/go-chi/cors
```

Immer wenn man ein neues Paket herunterlädt und verwendet muss man go mod vendor ausführen. Damit werden diese Pakete in den vendor Ordner verschoben - und den verwendet dann `go build` mit um das Executeable zu bauen.

Und hier der Code, der dazu führt, dass der HTTP Server mit dem chi Router gestartet wird und auf Requests auf dem PORT aus .env hört:

```golang
	router := chi.NewRouter()

	srv := &http.Server{ // srv ist also ein Pointer auf den http.Server - weil mit dem & die RAM-Adresse von http.Server in srv gespeichert wird
		Handler: router,
		Addr: ":"+portString,
	}

	log.Printf("Server starting on port %v\n",portString)
	err = srv.ListenAndServe() // hier bleibt der Code "stehen" - erst wenn ein Fehler auftritt, macht der Code weiter
	if err != nil {
		log.Fatal(err)
	}
```
