package main

import (
    "log"
    "net/http"
    "flag"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{} // use default options

func main() {
    ConnectToDatabase()

    //router.HandleFunc("/global/top/{amountOfTopPlayers}", GetTopPlayers).Methods("GET")
    //router.HandleFunc("/global/position/{playerID}", GetPosition).Methods("GET")
    //router.HandleFunc("/global/save_score", PutScore).Methods("PUT")

    //router.HandleFunc("/room/create", CreateRoom).Methods("GET")
    //router.HandleFunc("/room/connect/{roomID}", ConnectToRoom).Methods("POST")
    //router.HandleFunc("/room/disconnect/{roomID}", DisconnectFromRoom).Methods("POST")
    //router.HandleFunc("/room/{roomID}", GetRoomInfo).Methods("GET")

    //router.HandleFunc("/room/create", CreateRoom)


    http.HandleFunc("/room/create", CreateRoom)
    http.HandleFunc("/room/observe", Observe)
    http.HandleFunc("/room/join", JoinRoom)


    log.Fatal(http.ListenAndServe(*addr, nil))
}
