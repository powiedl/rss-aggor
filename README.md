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

# vendor Folder

Der vendor Folder ist vergleichbar mit node_modules in Javascript Projekten, aber in Go hat man üblicherweise viel weniger Dependencies, daher ist dieser Folder im Normalfall wesentlich kleiner. Daher kann man ihn im Normalfall (außer man hat wirklich sehr viele Dependencies) mit commiten (d. h. im Normalfall nimmt man in Go den vendor Folder nicht in .gitignore auf).

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

# HTTP Server schickt JSON Antworten

Da wir einen API Server bauen ist es üblich, dass dieser seine Antworten alle als JSON Objekte sendet. Als erstes erstellen wir uns eine Hilfsfunktion, die übergebene Daten in ein JSON konvertiert und dieses in einen http.ResponseWriter schreibt:

## Hilfsfunktion, die eine gegebene payload als JSON sendet

```golang
import (
	"encoding/json"
	"log"
	"net/http"
)

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
```

Das einzig "auffällige" an diesem Code ist, dass es ein leeres Interface als Payload entgegennimmt, d. h. "alles" (weil wir an dieser Stelle nicht genauer wissen, was die möglichen Payloads sind).

Die Methode `.WriteHeader()` des `http.ResponseWriter`s setzt den Statuscode, mit `.Header().Add()` kann man einzelne HTTP-Header setzen und mit `.Write()` schickt man schließlich die zu sendenden Daten. Falls das Konvertieren der payload in ein JSON funktioniert hat, sendet man den übergebenen `code` als Status Code, wenn das nicht funktioniert hat, sendet man ein 500 (Server Error).

## Eine handle Funktion

```golang
import "net/http"

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, 200, struct{}{})
}
```

Die Handlerfunktion muss immer diese Signatur aufweisen, d. h. als erstes Parameter muss sie einen http.ResponseWriter entgegennehmen und als zweites einen Zeiger auf einen http.Request.

## Die handleFunktion mit einer Route verbinden

Damit die handleFunktion auch ausgeführt wird, muss man sie jetzt noch mit einer Route verbinden. Das machen wir wieder in main.go mit diesen Befehlen

```golang
	v1Router := chi.NewRouter()
	v1Router.HandleFunc("/ready",handlerReadiness)
	router.Mount("/v1",v1Router)
```

Wir verwenden einen eigenen Router, damit wir ihn mit einem eigenen Pfad in der URL verbinden können. Damit können wir später eine v2 unserer API implementieren und diese als /v2 mounten.

Der gesamte Pfad, den man aufrufen muss, damit man zum handlerReadiness kommt lautet daher `/v1/ready`. Das `/v1` kommt vom Mount des v1Routers, das `/ready` kommt vom angegebenen PFad unserer HandleFunc.

Die obige Implementierung hat nur einen kleinen Schönheitsfehler - sie hört nicht nur auf GET sondern auch auf alle anderen HTTP-Verben (weil wir die `.HandleFunc` Methode verwenden). Wenn man - so wie meistens - auf ein bestimmtes HTTP Verb hören will, muss man stattdessen die gleichlautende Methode (in Go Notation, d. h. erster Buchstabe in Großbuchstaben) verwenden, d. h. `v1Router.Get("/healthz", handleReadiness)` (ja, wir haben den Pfad des Endpunktes zwischenzeitlich auch geändert, damit er der Kubernetes Konvention entspricht).

# Einheitlicher Error Response für unsere API

Wir erstellen uns eine Hilfsfunktion, die Fehler in einer einheitlichen Art und Weise zurückmeldet. Die Struktur der Rückmeldung soll (als JSON) soll so aussehen: `{"error":"Die übergebene Fehlermeldung"}`.

Und so sieht eine mögliche respondWithError Implementierung aus:

```golang
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
```

Interessant ist das `json: "error"`. Das reicht aus, damit der Marshaller weiß, dass er diese Information im JSON in ein Feld error schreiben soll. Diese "Annotation" funktioniert übrigens in beide Richtungen (was beim Error nicht notwendig ist), aber wenn wir später einen JSON-Body parsen werden die gleichen Annotationen verwendet, damit der Marshaller weiß, in welches Feld welche Information vom JSON geschrieben werden soll. Damit ist der Datenimport und export sehr einfach und mit wenig Aufwand möglich.

Und auch hier brauchen wir wieder eine handlerFunktion:

```golang
func handlerErr(w http.ResponseWriter, r *http.Request) {
	respondWithError(w,400,"Something went wrong")
}
```

Und wir müssen den Handler an eine Route binden (`v1Router.Get("/err",handlerErr)`).

# Datenbank

Als Datenbank verwenden wir PostgreSQL. Ich habe meine Datenbank auf meinem Raspberry laufen.

Dann installieren wir uns zwei Tools, damit wir einfacher mit SQL (aus unserem Go Programm) interagieren können:

```sh
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Datenbank Migration (mit Goose)

Bezeichnet die Operationen, um die Datenbank(struktur) so herzurichten, dass sie zu dem Programm "passt". Sehr vereinfacht hat gesprochen hat jede Datenbank Migrationen zwei Statements - ein "up" und ein "down" Statement. Wenn das "up" Statement eine Tabelle anlegt, soll/muss das dazugehörende "down" Statement diese Tabelle wieder löschen. Die Idee ist, dass man das up und das down Statement beliebig oft hintereinander ausführen kann und die Datenbank danach genauso wie davor aussieht.

Goose arbeitet die Migrationen in der Reihenfolge des Dateinamens ab, d. h. man beginnt jeden Dateinamen mit einer entsprechenden Sequenznummer. Die "up" Statements werden in aufsteigender Reihenfolge ausgeführt, die "down" Statements in absteigender Reihenfolge.

Außerdem interpretiert Goose die Kommentare in dem jeweiligen SQL-File (`+goose Up` damit weiß goose, dass jetzt die `up` Statements kommen bzw. `+goose Down` für die `down` Statements).

Die Migrationen selbst werden dann auf der Commandline mit `goose postgres postgres://.../rss-aggor-db up` bzw. `... down` ausgeführt. `postgres://.../rss-aggor-db` ist dabei durch den vollständigen Connectionstring zu ersetzen. Diesen sollte man sich auch in `.env` als `DB_URL` eintragen (damit man dann auch aus dem Go Programm darauf Zugriff hat).

## Datenbank Abfragen (mit Sqlc)

Sqlc wird mit einer YAML-Datei im Root Folder des Programms konfiguriert:

```yaml
version: '2'
sql:
  - schema: 'sql/schema'
    queries: 'sql/queries'
    engine: 'postgresql'
    gen:
      go:
        out: 'internal/database'
```

`queries` verweist auf den Ort, wo die SQL Queries gespeichert sind (bei uns `/sql/queries`). Sqlc liest diese Dateien und erstellt daraus "passende" Go Funktionen. Dabei wird in einem Kommentar über dem SQL Befehl die Go Funktion "beschrieben":

```sql
-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, name) VALUES ($1, $2, $3, $4) RETURNING *;
```

Sqlc generiert daraus eine Funktion `CreateUser`, die einen Datensatz zurückliefert und die vier Parameter hat. Der erste Parameter der Funktion wird im SQL an der Stelle $1 verwendet, der zweite an der Stelle $2 usw. Die Parameter der Funktion werden auch richtig typisiert (weil Sqlc die Struktur der Tabelle "versteht" und daher weiß, welcher Datentyp in dem entsprechenden SQL Datentyp gespeichert werden kann).

Mit `RETURNING *` gibt man an, dass der erzeugte Datensatz auch dem Aufrufer zurückgegeben werden soll (und diesen Datensatz liefert dann eben die von Sqlc generierte Funktion zurück).

Auf der Kommandozeile im Root-Verzeichnis (weil dort das sqlc.yaml liegt) ruft man dann `sqlc generate` auf. Das erzeugt den notwendigen Go Code. Dazu liest es eben die queries und das schema (damit es die Struktur "versteht") und speichert das Ergebnis unter out.

Dieser Code wird **niemals** manuell verändert, er wird komplett von sqlc gemanaged. Als Programmierer ist man nur dafür verantwortlich, die entsprechenden SQL-Statements im queries Verzeichnis zu schreiben.

## Datenbank im Go Code verwenden

### Einrichtung

Dazu muss man in `main.go` eine entsprechende apiConfig als struct mit dem Attribut DB anlegen. Dieses ist ein Zeiger auf `database.Queries`. Und `database` wiederum ist ein `import` aus dem out-Verzeichnis (`"github.com/powiedl/rss-aggor/internal/database"`).

```golang
dbUrl := os.Getenv("DBURL")
if dbUrl =="" {
	log.Fatal("DB_URL is not found in the environment")
}
conn, err := sql.Open("postgres",dbUrl) // sql kommt aus dem Standardpackage database.sql
if err != nil {
	log.Fatal("Can't connect to database:",err)
}

apiCfg := apiConfig {
	DB: database.New(conn),
}
```

Leider muss man manuell noch die **lib/pq** einbinden. Dazu muss man in der Kommandozeile `go get github.com/lib/pq` ausführen und in den imports von main.go dieses Package importieren (`_ "github.com/lib/pq"`). Der \_ ist wichtig, weil er Go sagt, dass es den Code im Programm einbinden muss, obwohl man es nicht direkt aufruft.

Die Datenbankconnection muss man dann noch in ein database.Queries "objekt" umgewandelt werden, weil es das ist, was die `apiConfig` als DB erwartet.

### http-Handler für Datenbankoperationen

Der Handler wird als Methode der apiConfig implementiert. Damit bleibt die Funktionssignatur des Handlers unverändert (das verlangt Go) und wir können trotzdem die zusätzlich notwendigen Informationen an den Handler transferieren

```golang
func (apiCfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, 200, struct{}{})
}
```

Wir definieren die Struktur des erwarteten Requests (der als Body des POST-Request geschickt wird) und initialisieren ein leeres params struct. Bei der Definition des struct verwenden wir wieder die json "Mapping" Beschreibung (das Attribut `Name` wird auf das JSON-Element `name` gemappt). Und diesmal werden wir daran interessiert sein, es aus dem Body des Requests zu extrahieren.

```golang
type parameters struct {
	Name string `json:"name"`
}
params := parameters{}
```

Das Extrahieren der Daten aus dem Request Body geschieht mit diesen Zeilen:

```golang
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Error parsing JSON: %v",err))
		return
	}
```

Danach erzeugen wir den Datensatz für den neuen User in der Datenbank und liefern den neuen User als Antwort auf den POST Request:

```golang
	user, err := apiCfg.DB.CreateUser(r.Context(),database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: params.Name,
	})

	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Couldn't create user:%v",err))
		return
	}

	respondWithJSON(w, 200, user)
```

Was ich im Moment noch nicht verstehe, warum der `user, err :=` keinen Fehler ergibt, weil es ja err bereits gibt ...

Und in `main.go` binden wir den Handler dann an die entsprechende Route ein: `v1Router.Post(apiCfg.handlerCreateUser)`.

Im folgenden der gesamte Code für den CreateUser Handler (aus Gründen der Übersichtlichkeit):

```golang
func (apiCfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Error parsing JSON: %v",err))
		return
	}
	user, err := apiCfg.DB.CreateUser(r.Context(),database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: params.Name,
	})

	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Couldn't create user:%v",err))
		return
	}

	respondWithJSON(w, 200, user)
}
```

### Verbesserung: Eigenes Schema für die Api Response

Im Moment verwenden wir das Schema, dass uns die Datenbank vorgibt für die Kommunikation mit dem Konsumenten unserer API, aber dieses Schema "gefällt" uns nicht. Daher definieren wir ein Schema für die API. Dazu machen wir ein File `model.go` in dem wir die API Modelle definieren (weil wir in `internal/database/models.go` keine Änderungen vornehmen dürfen).

Das beinhaltet eben das Modell für die API und eine entsprechende Konverter-Funktion zwischen dem Datenbank und dem API Modell.

```golang
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string `json:"name"`
}

func databaseUserToUser(dbUser database.User) User {
	return User{
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Name: dbUser.Name,
	}
}
```

# Authorization

Wir verwenden einfache APIKeys für unsere Applikation. Und als APIKey verwenden wir eine 64 Zeichen lange Zeichenkette. Dazu müssen wir die Tabelle Users um eine Spalte api_key erweitern. Dazu müssen wir eine neue Migration 002_users_apikey.sql anlegen. Man sollte niemals eine bestehende Migration verändern, wenn die Applikation bereits "released" ist, sondern eine neue Migration schreiben, die die Datenbank entsprechend adaptiert. In unserem Fall wollen wir eben eine Spalte api_key hinzufügen. Diese soll VARCHAR(64) NOT NULL UNIQUE sein. Damit wir das - wenn schon Datensätze existieren - gefahrlos machen können, müssen wir auch einen entsprechenden Default Wert setzen (der dann bei bereits bestehenden Records in der Tabelle verwendet wird):

```sql
ALTER TABLE users ADD COLUMN api_key VARCHAR(64) UNIQUE NOT NULL DEFAULT (
  encode(sha256(random()::text::bytea),'hex')
);
```

Der Defaultvalue wird als zufällige Zeichenfolge, die in ein byte Array konvertiert wird ermittelt. Dieses Bytearray dient dann als Parameter für die sha256 Funktion. Und dieser Hash wird als Hexadezimalzahl codiert (das sind dann genau 64 Zeichen).

In `sql/queries/users.sql` müssen wir das INSERT Statement für die Users Tabelle noch so erweitern, dass dort ebenfalls der API-Key generiert wird (damit auch neue User, die ab sofort angelegt werden, einen API Key bekommen) - eigentlich müssten wir nicht, weil sich der Default Value in der Tabellendefinition darum kümmert, dass ein API Key angelegt wird (aber ich lasse es so, damit ich dem Video weiter 1:1 folgen kann).

## Eigenes Auth Package

Als nächstes erstellen wir uns ein auth Package - auch wenn das nur eine einzige Funktion, nämlich GetAPIKey hat. Diese Funktion ermittelt den API-Key aus dem Header des Requests. Wir verlangen, dass der API Key in einem Header in dem Format übermittelt wird: `Authorization: ApiKey {insert apikey here}`

```golang
package auth

import (
	"errors"
	"net/http"
	"strings"
)

// GetAPIKey extracts an API Key from the headers of an HTTP request
// Example:
// Authorization: ApiKey {insert apikey here}
func GetAPIKey(headers http.Header) (string,error) {
	val := headers.Get("Authorization")
	if val == "" {
		return "",errors.New("no authentication info found")
	}
	vals := strings.Split(val," ")
	if len(vals) != 2 {
		return "", errors.New("malformed auth header")
	}
	if vals[0] != "ApiKey" {
		return "", errors.New("malformed first part of auth header")
	}
	return vals[0],nil
}
```

Der Code ist relativ einfach: Aus den Headers wird der Authorization Header extrahiert. Wenn er existiert wird er an den Leerzeichen aufgeteilt. Wenn damit genau zwei Worte gefunden werden und erste Wort "ApiKey" ist, wird das zweite Wort zurückgeliefert - andernfalls werden entsprechende Fehler zurückgemeldet.

## Auth Middleware

Nachdem wir mehrere Routen haben, die nur authentifiziert aufrufbar sind, erstellen wir eine entsprechende Middleware (in `middleware_auth.go` im Hauptverzeichnis). In dieser definieren wir einen Typ `authedHandler` (der zusätzlich zu den Dingen, die eine `http.HandlerFunc` als Parameter hat) noch einen weiteren Parameter hat.

Außerdem müssen wir eine Funktion erstellen, die diesen `authedHandler` wieder in eine `http.HandlerFunc` "zurückverwandelt". Diese Funktion wird als Closure gebaut (die eben eine "konvorme" `http.HandlerFunc` Funktion zurückliefert).

Hier ist der Code dieser Middleware (die im wesentlichen eben der Inhalt des handlerGetUser ist bzw. war - weil wir ihn dort ausschneiden):

```golang
func (apiCfg *apiConfig) middlewareAuth(handler authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil {
			respondWithError(w,403,fmt.Sprintf("Auth error:%v",err))
			return
		}
		fmt.Printf("Looking for user with API Key '%v'",apiKey)

		user, err := apiCfg.DB.GetUserByAPIKey(r.Context(),apiKey)
		if err != nil {
			respondWithError(w,400,fmt.Sprintf("Couldn't get user: %v",err))
			return
		}
		handler(w, r, user)
	}
}
```

Am Ende ruft die Middleware den übergebenen authedHandler auf und übergibt eben den aktuellen User. Damit das funktioniert, müssen wir die Signatur dieses Handlers (`handlerGetUser`) um den dritten Parameter - `user` vom Typ `database.User` erweitern). Dieser Handler muss dann nur noch die Daten entsprechend zurückliefern.

Jetzt haben wir aber ein Problem in `main.go`, weil der `handlerGetUser` jetzt nicht mehr der Signatur einer `http.HandlerFunc` entspricht - aber diese beim Einbinden der Route erforderlich ist. Zum "Glück" haben wir die Middleware, die genau diese Umwandlung macht, d. h. die Zeile in `main.go` muss wie folgt korrigiert werden: `v1Router.Get("/users",apiCfg.middlewareAuth(apiCfg.handlerGetUser))`.

# Parameter aus dem Pfad eines Requests auslesen (und nicht aus dem Body)

Wenn man einen Parameter aus dem Pfad eines Requests auslesen will, so muss man dazu wie in diesem Kapitel beschrieben vorgehen.

Bei der Definition der Route muss man an der richtigen Stelle den Parameter in `{}` angeben, z. b. `v1Router.Delete("/feed_follows/{feedFollowID}",apiCfg.middlewareAuth(apiCfg.handlerDeleteFeedFollows))` (in diesem Fall `{feedFollowID}`).

In der Handler Funktion kann man dann mit der Methode URLParam des chi Objekts diesen Parameter auslesen: `feedFollowID := chi.URLParam(r,"feedFollowID")`

Wichtig dabei ist, dass der 2. Parameter der Methode URLParam genauso geschrieben wird, wie man ihn in den geschwungenen Klammern angegeben hat.

# RSSFeed parsen

In `rss.go` ist die Funktionalität zum Parsen eines RSSFeeds enthalten. RSSFeeds sind im wesentlichen ein XML-Dokument mit einer bekannten Struktur. Und die Behandlung von XML in Go ist ähnlich wie die Behandlung von JSON. Es gibt auch wieder eine (Un)marshal-Methode, die ein Byte-Array entgegennimmt und entsprechend strukturierte Daten zurückliefert. Und das Mapping zwischen Attributnamen in Go und den Elementnamen im XML macht man mit `xml:"xmlattributname"` bei dem jeweiligen Attribut in der Definition des `struct`.

# "Taktgeber" in Go

Die Go Standardbibliothek time hat eine Methode NewTicker. Diese erwartet einen Parameter - den zeitlichen Abstand zwischen zwei Takten (als time.Duration). Immer wenn time.Duration vergangen ist, feuert der zugehörige Channel. Mit einer einfachen for-Schleife kann man dadurch leicht etwas auslösen:

```golang
ticker := time.NewTicker(time.Second)
s:="Tik"
for ; ; <-ticker.C {
	fmt.Println(s)
	if s == "Tik" {
		s = "Tak"
	} else {
		s = "Tik"
	}
}
```

Dieses Codesnippet schreibt abwechselnd jede Sekunde Tik, dann Tak, dann Tik, ...

Dadurch, dass sowohl die Initilisierung als auch die Bedingung leer sind wird as erste Tik sofort geschrieben. Wenn man dort etwas angeben würde, würde erst nach einer Sekunde mit der Ausgabe begonnen werden (weil wir das ja im AFTER Teil der for Schleife definiert haben)

# Synchronisation von Go Routinen mit Waitgroup

In der Standardlibrary sync gibt es sogenannte Waitgroups. Diese dienen dazu verschiedene Go Routinen miteinander zu synchronisieren bzw. in der "aufrufenden" Routine darauf zu warten, dass alle fertig geworden sind. Dazu definiert man eine Waitgroup `wg := &sync.WaitGroup{}`. Unmittelbar vor jedem Aufruf einer Go Routine ruft man `wg.Add(1)` auf. In den Go Routinen schreibt man als erste Zeile `wg.Done()`. Und in der aufrufenden Funktion kann man dann an der gewünschten Stelle ein `wg.Wait()` aufrufen. An genau dieser Stelle wird gewartet, bis alle go Routinen beendet wurden (womit jede Go Routine das `wg.Done()` aufgerufen hat). Wenn das der Fall ist, macht die aufrufende Routine mit der Zeile nach dem `wg.Wait()` weiter. `wg.Wait()` blockiert also die Ausführung der Routine bis alle Go Routinen fertig geworden sind.
