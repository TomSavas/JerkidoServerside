package main

import (
	"net/http"
	"time"
)

func main() {
	err := ConnectToDatabase()

	go func() {
		for {
			LogInfo("RoomDB", "Periodically removing empty rooms.")
			RemoveEmptyRooms()
			time.Sleep(3600 * 1000 * time.Millisecond)
		}
	}()

	Fatal(err, "Failed connecting to the database")

	http.HandleFunc("/room/create", CreateRoom)
	http.HandleFunc("/room/observe", Observe)
	http.HandleFunc("/room/join", Join)

	Fatal(http.ListenAndServe("0.0.0.0:8080", nil), "Failed running a webserver")
}
