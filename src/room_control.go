package main

import (
	"fmt"
	"net/http"
	"time"
)

func ReadConnectingUser(connection *WSConnection) *Player {
	var user Player
	err := connection.ReadJSON(&user)
	Fatal(err, "Failed to read the connecting user")

	return &user
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	connection := ToWSConnection(w, r, nil)
	owner := ReadConnectingUser(connection)

	owner.SaveScore()
	room := NewRoom(owner.ID, owner.IsPlayer, 5)
	if owner.IsPlayer {
		LogInfo("Room", "Created a new room: "+room.ID+", owner: "+room.OwnerID+". Owner is also a player.")
	} else {
		LogInfo("Room", "Created a new room: "+room.ID+", owner: "+room.OwnerID+".")
	}
	room.Write()

	connection.WriteJSON(*room.Info())

	if owner.IsPlayer {
		ControlRoom(w, r, connection, room, owner, PlayGame)
	} else {
		ControlRoom(w, r, connection, room, owner, ObserveRoom)
	}
}

func ControlRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player, onPlay func(http.ResponseWriter, *http.Request, *WSConnection, *Room, *Player)) {
	playRequestedChan := make(chan interface{}, 1)

	go WaitForRoomStateUpdate(connection, player, playRequestedChan)

	for {
		if connection.IsClosed() {
			LogInfo("Room "+room.ID, "Connection closed by the room owner: "+room.OwnerID+".")
			DisconnectPlayer(player, room, connection)

			return
		}

		// Send updates about the room until the game is started
		startedGame := false
		go func() {
			for !startedGame && !connection.IsClosed() {
				ObserveRoom(w, r, connection, room, player)
			}
		}()

		if hasValue, playRequested := ChannelHasValueWithTimeout(playRequestedChan, 3600); hasValue && playRequested.(bool) {
			// Ensure that the state is changed into Transition before launching onPlay
			room.ChangeRoomState(Transition)
			// Flag game as started to stop ObserveRoom
			startedGame = true

			go func() {
				time.Sleep(800 * time.Millisecond)
				room.ChangeRoomState(CountingDown_3)
				time.Sleep(time.Second)
				room.ChangeRoomState(CountingDown_2)
				time.Sleep(time.Second)
				room.ChangeRoomState(CountingDown_1)
				time.Sleep(time.Second)
				room.ChangeRoomState(CountingDown_0)
				time.Sleep(700 * time.Millisecond)
				room.ChangeRoomState(Play)
				time.Sleep(time.Duration(room.PlayTimeInSeconds*1000) * time.Millisecond)
				room.ChangeRoomState(End)
			}()
			onPlay(w, r, connection, room, player)

			go WaitForRoomStateUpdate(connection, player, playRequestedChan)
		} else {
			LogInfo("Room "+room.ID, "ControlRoom timed out for owner: "+room.OwnerID+". Removing the owner.")
			DisconnectPlayer(player, room, connection)

			return
		}
	}
}

func WaitForRoomStateUpdate(connection *WSConnection, player *Player, playRequested chan interface{}) {
	var room Room
	writingFailureCount := 0

	for {
		if connection.IsClosed() || writingFailureCount > 10 {
			playRequested <- false
			return
		}

		err := connection.ReadJSON(&room)
		if err != nil {
			Log(err, "Room "+room.ID, "Failed to read room state change from the owner: "+player.ID+". ")
			writingFailureCount++
		}

		if room.State == Play {
			playRequested <- true
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func Join(w http.ResponseWriter, r *http.Request) {
	connection := ToWSConnection(w, r, nil)
	player := ReadConnectingUser(connection)

	room := GetExistingRoom(player.RoomID)
	if !Exists(room.PlayerIDs, player.ID) {
		player.SaveScore()

		room.AddPlayer(player.ID)

		LogInfo("Room "+player.RoomID, "Adding player: "+player.ID+" to the room. Player list: "+fmt.Sprintf("%+v", room.PlayerIDs))
	}

	for !connection.IsClosed() {
		PlayGame(w, r, connection, room, player)
	}
	room.RemovePlayer(player.ID)
	connection.Close()
}

func Observe(w http.ResponseWriter, r *http.Request) {
	connection := ToWSConnection(w, r, nil)

	observer := ReadConnectingUser(connection)
	room := GetExistingRoom(observer.RoomID)

	for !connection.IsClosed() {
		ObserveRoom(w, r, connection, room, observer)
	}
}

func ObserveRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player) {
	writingFailureCount := 0
	if connection.IsClosed() {
		LogInfo("Room "+room.ID, "Observer disconnected: "+player.ID+".")
		room.RemovePlayer(player.ID)
		connection.Close()

		return
	}

	room.Read()

	err := connection.WriteJSON(*room.Info())
	if err != nil {
		writingFailureCount++
	}

	if writingFailureCount > 10 {
		Log(err, "Room "+room.ID, "Failed sending room info to the observer: "+player.ID+" 10 times. Removing the player.")

		connection.Close()
		return
	} else {
		Log(err, "Room "+room.ID, "Failed sending room info to the observer: "+player.ID)
	}

	time.Sleep(100 * time.Millisecond)
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player) {
	writingFailureCount := 10
	player.Online = true

	room.Read()
	connection.WriteJSON(*room.Info())

	for {
		if connection.IsClosed() || writingFailureCount > 10 {
			LogInfo("Room "+room.ID, "Player disconnected: "+player.ID)
			DisconnectPlayer(player, room, connection)

			return
		}

		room.Read()
		err := connection.WriteJSON(room.Info())
		Log(err, "Room "+room.ID, "Failed sending room info before play state to player: "+player.ID)
		if err != nil {
			writingFailureCount++
		}

		if room.State == Play {
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	for {
		if connection.IsClosed() {
			LogInfo("Room "+room.ID, "Player disconnected: "+player.ID)
			DisconnectPlayer(player, room, connection)

			break
		}

		err := connection.ReadJSON(&player)
		Log(err, "Room "+room.ID, "Failed reading player score: "+player.ID)
		player.SaveScore()

		room.Read()
		if room.State == End {
			connection.WriteJSON(*room.Info())

			break
		}
	}
}
