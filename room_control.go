package main

import (
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
    room := NewRoom(owner.ID, owner.IsPlayer)
    LogInfo("Room", "Created a new room: " + room.ID + ", owner: " + room.OwnerID + ".")
    room.Write()

    roomInfo := room.Info()
    connection.WriteJSON(roomInfo)

    if owner.IsPlayer {
        ControlRoom(w, r, connection, room, owner, PlayGame)
    } else {
        ControlRoom(w, r, connection, room, owner, ObserveRoom)
    }
}

func ControlRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player, onPlay func(http.ResponseWriter, *http.Request, *WSConnection, *Room, *Player)) {
    roomStateChanged := make(chan interface{}, 1)
    go WaitForRoomStateUpdate(connection, roomStateChanged)

	for {
        if connection.IsClosed() {
            LogInfo("Room " + room.ID, "Connection closed by the room owner: " + room.OwnerID + ".")

            return
        }

        roomUpdated := room.ReadWithStatusReport()

		roomInfo := room.Info()

		if hasValue, _ := ChannelHasValue(roomStateChanged); hasValue {
            go func() {
                room.ChangeRoomState(Transition)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomState(CountingDown_3)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomState(CountingDown_2)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomState(CountingDown_1)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomState(CountingDown_0)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomState(Play)
		    }()
            onPlay(w, r, connection, room, player)
            break
		}

        if roomUpdated {
            err := connection.WriteJSON(*roomInfo)
            Log(err, "Failed writing room to the controlling user")
        }

		time.Sleep(100 * time.Millisecond)
	}

}

func WaitForRoomStateUpdate(connection *WSConnection, roomStateChanged chan interface{}) {
	var room Room

	for {
        if connection.IsClosed() {
            return
        }

		err := connection.ReadJSON(&room)
        Log(err, "Room " + room.ID, "Failed to read room state change from the owner.")

		if room.State != WaitingForPlayers {
			roomStateChanged <- true
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
		room.PlayerIDs = append(room.PlayerIDs, player.ID)
        room.Write()
	}

	PlayGame(w, r, connection, room, player)
}

func Observe(w http.ResponseWriter, r *http.Request) {
    connection := ToWSConnection(w, r, nil)

    observer := ReadConnectingUser(connection)
    room := GetExistingRoom(observer.RoomID)

    ObserveRoom(w, r, connection, room, observer)
}

func ObserveRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player) {
    // If there is no Read method active, the closeHandler is not triggered...
    // This doesn't seem like the desired behaviour. Investigate.
    go func() {
        for {
            _, _, err := connection.ReadMessage()

            if connection.IsClosed() || err != nil {
                LogInfo("Room " + room.ID, "Observer disconnected: " + player.ID + ".")

                return
            }
        }
    }()

    for {
        if connection.IsClosed() {
            LogInfo("Room " + room.ID, "Observer disconnected: " + player.ID + ".")

            return
        }

        room.Read()
        roomInfo := room.Info()

        err := connection.WriteJSON(*roomInfo)
        Log(err, "Cannot write room info to the observer")

        time.Sleep(100 * time.Millisecond)
    }
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player) {
	player.Online = true

    roomInfo := room.Info()
	connection.WriteJSON(*roomInfo)

	previousRoomState := WaitingForPlayers
	for {
		if connection.IsClosed() {
            LogInfo("Room " + room.ID, "Player disconnected: " + player.ID)

			player.Online = false
            player.SaveScore()
			return
		}

        room.Read()

		if room.State != previousRoomState {
			previousRoomState = room.State
            _ = connection.WriteJSON(room.Info())

			if room.State == Play {
				break
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	for {
        if connection.IsClosed() {
            LogInfo("Room " + room.ID, "Player disconnected: " + player.ID)

            player.Online = false
            player.SaveScore()
            break
        }

		err := connection.ReadJSON(&player)
        Log(err, "Room " + room.ID, "Failed reading player score: " + player.ID)
        player.SaveScore()
	}
}
