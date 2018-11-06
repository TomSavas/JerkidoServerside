package main

import (
    "github.com/gorilla/mux"
    "github.com/globalsign/mgo"
    "log"
    "net/http"
)

var DB *mgo.Database;
func main() {
    session, err := mgo.Dial("localhost")
    if err != nil {
            panic(err)
    }
    defer session.Close()

    DB = session.DB("JerkidoData")

    router := mux.NewRouter()
    router.HandleFunc("/global/top/{amountOfTopPlayers}", GetTopPlayers).Methods("GET")
    router.HandleFunc("/global/position/{playerID}", GetPosition).Methods("GET")
    router.HandleFunc("/global/save_score", PutScore).Methods("PUT")

    router.HandleFunc("/room/create", CreateRoom).Methods("GET")
    router.HandleFunc("/room/connect/{roomID}", ConnectToRoom).Methods("POST")
    router.HandleFunc("/room/disconnect/{roomID}", DisconnectFromRoom).Methods("POST")
    router.HandleFunc("/room/{roomID}", GetRoomInfo).Methods("GET")

    log.Fatal(http.ListenAndServe(":8000", router))
}
