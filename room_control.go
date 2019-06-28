package main

import (
    "net/http"
    "time"
    "fmt"
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
        LogInfo("Room", "Created a new room: " + room.ID + ", owner: " + room.OwnerID + ". Owner is also a player.")
    } else {
        LogInfo("Room", "Created a new room: " + room.ID + ", owner: " + room.OwnerID + ".")
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
    playRequested := make(chan interface{}, 1)

    go WaitForRoomStateUpdate(connection, playRequested)

	for {
        if connection.IsClosed() {
            LogInfo("Room " + room.ID, "Connection closed by the room owner: " + room.OwnerID + ".")

            return
        }

        // Send updates about the room until the game is started
        startedGame := false
        go func() {
            for !startedGame && !connection.IsClosed() {
                ObserveRoom(w, r, connection, room, player)
            }
        }()

		if hasValue, _ := ChannelHasValueWithTimeout(playRequested, 3600); hasValue {
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
                time.Sleep(time.Duration(room.PlayTimeInSeconds * 1000) * time.Millisecond)
                room.ChangeRoomState(End)
                time.Sleep(200 * time.Millisecond)
                room.ChangeRoomState(WaitingForPlayers)
		    }()
            onPlay(w, r, connection, room, player)

            go WaitForRoomStateUpdate(connection, playRequested)
		} else {
            LogInfo("Room " + room.ID, "ControlRoom timed out for owner: " + room.OwnerID + ". Removing the owner.")

            connection.Close();
            return
        }
    }
}

func WaitForRoomStateUpdate(connection *WSConnection, playRequested chan interface{}) {
	var room Room
    writingFailureCount := 0

	for {
        if connection.IsClosed() || writingFailureCount > 10 {
            return
        }

		err := connection.ReadJSON(&room)
        if err != nil {
            Log(err, "Room " + room.ID + "(WaitingForRoomStateUpdate)", "Failed to read room state change from the owner.")
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
		room.PlayerIDs = append(room.PlayerIDs, player.ID)
        room.Write()

        LogInfo("Room " + player.RoomID, "Adding player: " + player.ID + " to the room. Player list: " + fmt.Sprintf("%+v", room.PlayerIDs))
	}

    for !connection.IsClosed() {
        PlayGame(w, r, connection, room, player)
    }
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
        LogInfo("Room " + room.ID, "Observer disconnected: " + player.ID + ".")

        return
    }

    room.Read()

    err := connection.WriteJSON(*room.Info())
    if err != nil {
        writingFailureCount++
    }

    if writingFailureCount > 10 {
        Log(err, "Room " + room.ID, "Failed sending player score: " + player.ID + " 10 times. Removing the player.")

        connection.Close();
        return
    } else {
        Log(err, "Room " + room.ID, "Failed sending player score: " + player.ID)
    }

    time.Sleep(100 * time.Millisecond)
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, player *Player) {
	player.Online = true

    room.Read()
	connection.WriteJSON(*room.Info())

    for {
        if connection.IsClosed() {
            LogInfo("Room " + room.ID, "Player disconnected: " + player.ID)

            player.Online = false
            player.SaveScore()
            return
        }

        room.Read()
        err := connection.WriteJSON(room.Info())
        Log(err, "Room " + room.ID, "Failed sending room info before play state to player: " + player.ID)

        if room.State == Play {
            break
        }

        time.Sleep(200 * time.Millisecond)
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


        room.Read()
        if room.State == End {
            connection.WriteJSON(*room.Info())

            break
        }
    }
}
