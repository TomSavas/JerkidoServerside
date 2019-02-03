package main

import (
    "net/http"
)

func main() {
    err := ConnectToDatabase()
    Fatal(err, "Failed connecting to the database")

    http.HandleFunc("/room/create", CreateRoom)
    http.HandleFunc("/room/observe", Observe)
    http.HandleFunc("/room/join", Join)

    Fatal(http.ListenAndServe("0.0.0.0:8080", nil), "Failed running a webserver")
}
