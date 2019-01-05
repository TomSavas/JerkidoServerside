package main

import (
    "log"
    "net/http"
    "flag"
	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func main() {
    //For testing purposes ignore cross-site request forgery issues
    upgrader.CheckOrigin = func (r *http.Request) bool {
        return true
    }

    router := mux.NewRouter()

    ConnectToDatabase()
    router.HandleFunc("/global/top/{amountOfTopPlayers}", GetTopPlayers).Methods("GET")
    //router.HandleFunc("/global/position/{playerID}", GetPosition).Methods("GET")
    //router.HandleFunc("/global/save_score", PutScore).Methods("PUT")

    http.HandleFunc("/room/create", CreateRoom)
    http.HandleFunc("/room/observe", Observe)
    http.HandleFunc("/room/join", JoinRoom)


    srv := &http.Server{
        Handler:      router,
        Addr:         "localhost:8081",//addr2.String(),
        // Good practice: enforce timeouts for servers you create!
        // WriteTimeout: 15 * time.Second,
        // ReadTimeout:  15 * time.Second,
    }

    go func() {log.Fatal(http.ListenAndServe(*addr, nil))} ()
    log.Fatal(srv.ListenAndServe())
}
