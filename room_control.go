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
    room := NewRoom(owner.ID, owner.IsPlayer)
    //fmt.Println("Created a new room: " + room.ID + " owner: " + room.OwnerID)
    room.Write()

    roomInfo := room.Info()
    connection.WriteJSON(roomInfo)

    if owner.IsPlayer {
        ControlRoom(w, r, connection, room, PlayGame)
    } else {
        ControlRoom(w, r, connection, room, ObserveRoom)
    }
}

func ControlRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, onPlay func(http.ResponseWriter, *http.Request, *WSConnection, *Room)) {
    roomStateChanged := make(chan interface{}, 1)
    go WaitForRoomStateUpdate(connection, roomStateChanged)

	for {
        if connection.IsClosed() {
            fmt.Println("Connection closed by the observer")
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
            onPlay(w, r, connection, room)
            break
		}

        if roomUpdated {
            ClearTerminal()
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
        Log(err, "Failed to read room update")

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

	PlayGame(w, r, connection, room)
}

func Observe(w http.ResponseWriter, r *http.Request) {
    connection := ToWSConnection(w, r, nil)

    observer := ReadConnectingUser(connection)
    room := GetExistingRoom(observer.RoomID)

    ObserveRoom(w, r, connection, room)
}

func ObserveRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room) {
    // If there is no Read method active, the closeHandler is not triggered...
    // This doesn't seem like the desired behaviour. Investigate.
    go func() {
        for {
            if connection.IsClosed() {
                fmt.Println("Connection closed by the observer")
                return
            }
            _, _, _ = connection.ReadMessage()
        }
    }()

    for {
        if connection.IsClosed() {
            fmt.Println("Connection closed by the observer (in observe)")
            return
        }

        room.Read()
        roomInfo := room.Info()

        err := connection.WriteJSON(*roomInfo)
        Log(err, "Cannot write room info to the observer")

        time.Sleep(100 * time.Millisecond)
    }
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room) {
	var player Player
	player.Online = true

    roomInfo := room.Info()
	connection.WriteJSON(*roomInfo)

	previousRoomState := WaitingForPlayers
	for {
		if connection.IsClosed() {
			fmt.Println("Player disconnected")
			player.Online = false
            (&player).SaveScore()
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
            fmt.Println("Player disconnected")
            player.Online = false
            (&player).SaveScore()
            break
        }

		err := connection.ReadJSON(&player)
        Log(err, "Failed to read the player")
        (&player).SaveScore()
	}
}
