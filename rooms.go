package main

import (
    //"github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "gopkg.in/mgo.v2/bson"

    //"log"
    "bytes"
    "fmt"
    "math/rand"
    "net/http"
    "sort"
    "time"

    "github.com/globalsign/mgo"
)

type Room struct {
    ID           string    `json:"id"`
    State        RoomState `json:"state"`
    PlayerIDs    []string  `json:"playerids"`
    ObeserverIDs []string  `json:"observerids"`
    OwnerID      string    `json:"ownerid"`
}

type RoomInfo struct {
    ID      string    `json:"id"`
    State   RoomState `json:"state"`
    Players []Player  `json:"players"`
}

type RoomState int

const (
    WaitingForPlayers RoomState = 0
    CountingDown_3    RoomState = 1
    CountingDown_2    RoomState = 2
    CountingDown_1    RoomState = 3
    CountingDown_0    RoomState = 4
    Play              RoomState = 5
    End               RoomState = 6
)

func GetRooms() *mgo.Collection {
    return GetCollection("Rooms")
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
connection, isConnectionClosed, err := UpgradeConnToWebSocketConn(w, r)

    var owner User
    err = connection.ReadJSON(&owner)
    if err != nil {
        fmt.Print("Error reading room creator. ")
        fmt.Println(err)
        return
    }

    rooms := GetRooms()
    room := Room{GenerateUniqueRoomCode(rooms), WaitingForPlayers, []string{}, []string{owner.ID}, owner.ID}
    fmt.Println("Created a new room: " + room.ID + " owner: " + room.OwnerID)

    err = rooms.Insert(&room)
    if err != nil {
        fmt.Println(err)
        return
    }
    connection.WriteMessage(websocket.TextMessage, []byte("{\"id\":\""+room.ID+"\"}"))

    if owner.IsPlayer {
        //JoinRoom(w, r, connection, isConnectionClosed, room.ID)
        ControlRoom(w, r, connection, isConnectionClosed, room.ID, JoinRoom)
    } else {
        //ObserveRoom(w, r, connection, isConnectionClosed, room.ID)
        ControlRoom(w, r, connection, isConnectionClosed, room.ID, ObserveRoom)
    }
}

func Observe(w http.ResponseWriter, r *http.Request) {
    connection, isConnectionClosed, err := UpgradeConnToWebSocketConn(w, r)

    var observer User
    err = connection.ReadJSON(&observer)
    if err != nil {
        fmt.Printf("Error reading observer: %+v: ", observer)
        fmt.Println(err)
        return
    }
    fmt.Println("Observer connected: " + observer.ID)

    ObserveRoom(w, r, connection, isConnectionClosed, observer.RoomID)
}

func ObserveRoom(w http.ResponseWriter, r *http.Request, connection *websocket.Conn, isConnectionClosed chan bool, roomID string) {
    var existingRoom Room
    rooms := GetRooms()
    err := rooms.Find(bson.M{"id": roomID}).One(&existingRoom)
    fmt.Println("Room: " + existingRoom.ID + " owner: " + existingRoom.OwnerID)

    //If there is no Read method active, the closeHandler is not triggered...
    go func() {
        for {
            if IsConnectionClosed(isConnectionClosed) {
                fmt.Println("Connection closed by the observer")
                return
            }

            _, _, _ = connection.ReadMessage()
        }
    }()

    maxOnline := 0
    for {
        someOnline := 0
        if IsConnectionClosed(isConnectionClosed) {
            fmt.Println("Connection closed by the observer (in observe)")
            return
        }

        err = rooms.Find(bson.M{"id": roomID}).One(&existingRoom)

        if err != nil {
            fmt.Print("Failed to read rooms: ")
            fmt.Println(err)
            break
        }

        players, errs := PlayersInRoom(&existingRoom)

        for _, err = range errs {
            if err != nil {
                fmt.Println(err)
                //return
            }
        }

        for _, player := range players {
            if player.Online {
                someOnline++
            }
        }

        if someOnline > maxOnline {
            maxOnline = someOnline
        }

        ClearTerminal()
        fmt.Printf("People online: %d\n", someOnline)
        fmt.Printf("Max people online: %d\n", maxOnline)
        switch existingRoom.State {
        case WaitingForPlayers:
            fmt.Println("Waiting for players...")
        case CountingDown_3:
            fmt.Println("Counting down 3...")
        case CountingDown_2:
            fmt.Println("Counting down 2...")
        case CountingDown_1:
            fmt.Println("Counting down 1...")
        case CountingDown_0:
            fmt.Println("Counting down 0...")
        case Play:
            sort.Slice(players, func(i, j int) bool { return players[i].Score > players[j].Score })

            fmt.Println("Room: " + roomID + " owner: " + existingRoom.OwnerID)
            for _, player := range players {
                fmt.Printf("ID: %s, score: %d, online: %t\n", player.ID, player.Score, player.Online)
            }
        case End:
            fmt.Println("Game ended")
        default:
            fmt.Println("Wrong game state")
        }

        roomInfo := RoomInfo{existingRoom.ID, existingRoom.State, players}

        err = connection.WriteJSON(roomInfo)
        if err != nil {
            if IsConnectionClosed(isConnectionClosed) {
                fmt.Println("Connection closed by the observer (in observe)")
                return
            } else {
                fmt.Println("Cannot write room info to the observer")
                fmt.Println(err)
            }
        }

        time.Sleep(100 * time.Millisecond)
    }
}

func ControlRoom(w http.ResponseWriter, r *http.Request, connection *websocket.Conn, isConnectionClosed chan bool, roomID string, onPlay func(http.ResponseWriter, *http.Request, *websocket.Conn, chan bool, string)) {
	var room Room
	rooms := GetRooms()

	err := rooms.Find(bson.M{"id": roomID}).One(&room)
	fmt.Println("Room: " + room.ID + " owner: " + room.OwnerID)

    roomStateChanged := make(chan interface{}, 1)
    if room.State == WaitingForPlayers {
        go WaitForRoomStateUpdate(connection, roomStateChanged)
    }

	for {
        if IsConnectionClosed(isConnectionClosed) {
            fmt.Println("Connection closed by the observer")
            return
        }

		err = rooms.Find(bson.M{"id": roomID}).One(&room)
		if err != nil {
			fmt.Print("Failed to read rooms: ")
			fmt.Println(err)
			break
		}

		players, errs := PlayersInRoom(&room)
		for _, err = range errs {
			if err != nil {
				fmt.Println("Error while trying to read players. ")
				fmt.Println(err)
				return
			}
		}

		if hasValue, _ := ChannelHasValue(roomStateChanged); hasValue {
            go func() {
                ChangeRoomState(room, CountingDown_3)
                time.Sleep(1000 * time.Millisecond)
                ChangeRoomState(room, CountingDown_2)
                time.Sleep(1000 * time.Millisecond)
                ChangeRoomState(room, CountingDown_1)
                time.Sleep(1000 * time.Millisecond)
                ChangeRoomState(room, CountingDown_0)
                time.Sleep(1000 * time.Millisecond)
                ChangeRoomState(room, Play)
		    }()
            onPlay(w, r, connection, isConnectionClosed, roomID)
            break
		}

		ClearTerminal()
		for _, player := range players {
			fmt.Printf("%s %d %t\n", player.ID, player.Score, player.Online)
		}

		time.Sleep(100 * time.Millisecond)
	}

}

func WaitForRoomStateUpdate(connection *websocket.Conn, roomStateChanged chan interface{}) {
	var room Room

	for {
		err := connection.ReadJSON(&room)
		if err != nil {
			fmt.Println("Failed to read room update. ")
			fmt.Print(err)
			return
		}

		if room.State != WaitingForPlayers {
			roomStateChanged <- true
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func ChangeRoomState(room Room, state RoomState) error {
	var err error = nil
	if room.State != state {
		fmt.Printf("Changing the state of the room from %d to %d", room.State, state)

		err = GetRooms().Update(
			bson.M{"id": room.ID},
			bson.M{"$set": bson.M{"state": state}},
		)
	}

	return err
}

func GenerateUniqueRoomCode(rooms *mgo.Collection) string {
	code := GenerateRoomCode(5)

	if (rooms.Find(bson.M{"ID": code}) == nil) {
		return GenerateUniqueRoomCode(rooms)
	}

	return code
}

func GenerateRoomCode(codeLength int) string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var code bytes.Buffer

	for i := 0; i < codeLength; i++ {
		if rng.Intn(2) == 0 {
			letter := rng.Intn(26)
			code.WriteString(string(letter + 65))
		} else {
			number := rng.Intn(10)
			code.WriteString(string(number + 48))
		}
	}

	return code.String()
}

func Join(w http.ResponseWriter, r *http.Request) {
	connection, isConnectionClosed, err := UpgradeConnToWebSocketConn(w, r)

	var user User
	err = connection.ReadJSON(&user)
	if err != nil {
		fmt.Print("Error reading connecting user: ")
		fmt.Println(err)
		return
	}

	rooms := GetRooms()
	if err != nil {
		fmt.Println("Failed reading rooms. ")
		fmt.Println(err)
		return
	}

	fmt.Println("looking for a room")
	var existingRoom Room
	err = rooms.Find(bson.M{"id": user.RoomID}).One(&existingRoom)

	if err != nil {
		fmt.Println("Failed geting room with id: " + user.RoomID)
		fmt.Println(err)
		return
	}

	if !Exists(existingRoom.PlayerIDs, user.ID) {
		SaveScore(&Player{user.ID, 0, 0, true}) //Don't leave this. It will override an existing player's top score.

		changedPlayerIDs := append(existingRoom.PlayerIDs, user.ID)
		err = rooms.Update(
			bson.M{"id": existingRoom.ID},
			bson.M{"$set": bson.M{"playerids": changedPlayerIDs}},
		)
		if err != nil {
			fmt.Println("Failed adding player to playerids in room: " + user.RoomID)
			fmt.Println(err)
		}
	}

	PlayGame(w, r, connection, isConnectionClosed, existingRoom.ID)
}

func JoinRoom(w http.ResponseWriter, r *http.Request, connection *websocket.Conn, isConnectionClosed chan bool, roomID string) {
	//connection, isConnectionClosed, err := UpgradeConnToWebSocketConn(w, r)
    var err error

//	var user User
//	err = connection.ReadJSON(&user)
//	if err != nil {
//		fmt.Print("Error reading connecting user: ")
//		fmt.Println(err)
//		return
//	}

	rooms := GetRooms()
	if err != nil {
		fmt.Println(err)
		return
	}

	var existingRoom Room
	err = rooms.Find(bson.M{"id": roomID}).One(&existingRoom)

	if err != nil {
		fmt.Println(err)
		return
	}

	if !Exists(existingRoom.PlayerIDs, existingRoom.OwnerID) {
		changedPlayerIDs := append(existingRoom.PlayerIDs, existingRoom.OwnerID)
		rooms.Update(
			bson.M{"id": existingRoom.ID},
			bson.M{"$set": bson.M{"playerids": changedPlayerIDs}},
		)
	}

	PlayGame(w, r, connection, isConnectionClosed, existingRoom.ID)
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *websocket.Conn, isConnectionClosed chan bool, roomID string) {
    //fmt.Println("playing...")
	var player Player
	player.Online = true

	connection.WriteJSON(RoomInfo{roomID, WaitingForPlayers, []Player{}})

	previousRoomState := WaitingForPlayers
	for {
		if IsConnectionClosed(isConnectionClosed) {
			fmt.Println("Player disconnected")
			player.Online = false
			SaveScore(&player)
			return
		}

		var room Room
		err := GetRooms().Find(bson.M{"id": roomID}).One(&room)

		if err != nil {
			fmt.Println(err)
			return
		}

		if room.State != previousRoomState {
			previousRoomState = room.State
			err = connection.WriteJSON(RoomInfo{room.ID, room.State, []Player{}})
			if room.State == Play {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	for {
		err := connection.ReadJSON(&player)

		if err != nil {
			if IsConnectionClosed(isConnectionClosed) {
				fmt.Println("Player disconnected")
				player.Online = false
				SaveScore(&player)
				break
			} else {
				fmt.Print("Error reading message: ")
				fmt.Println(err)
			}
		} else {
			SaveScore(&player)
		}

        // for debugging purposes
        ClearTerminal()
		var room Room
		err = GetRooms().Find(bson.M{"id": roomID}).One(&room)

        players, _ := PlayersInRoom(&room)
        sort.Slice(players, func(i, j int) bool { return players[i].Score > players[j].Score })

        fmt.Println("Room: " + roomID + " owner: " + room.OwnerID)
        for _, player := range players {
            fmt.Printf("ID: %s, score: %d, online: %t\n", player.ID, player.Score, player.Online)
        }
        // for debugging purposes
	}
}

/*
func DisconnectFromRoom(w http.ResponseWriter, r *http.Request) {
    rooms := GetRooms()
    roomID := mux.Vars(r)["roomID"]
    playerData, err := JsonToMap(r)
    if err != nil {
        log.Println(err)
        w.WriteHeader(400)
        return
    }

    var existingRoom Room
    err = rooms.Find(bson.M{"id":roomID}).One(&existingRoom)
    if err != nil {
        log.Println(err)
        w.WriteHeader(404)
        return
    }

    if Exists(existingRoom.PlayerIDs, playerData["id"].(string)) {
        var changedPlayerIDs []string

        for i := 0; i < len(existingRoom.PlayerIDs); i++ {
            if(existingRoom.PlayerIDs[i] != playerData["id"].(string)) {
                changedPlayerIDs = append(changedPlayerIDs, existingRoom.PlayerIDs[i])
            }
        }

        log.Println(changedPlayerIDs)
        rooms.Update(
            bson.M{"id":existingRoom.ID},
            bson.M{"$set": bson.M{"playerids":changedPlayerIDs}},
        )
        w.WriteHeader(200)
    } else {
        w.WriteHeader(404)
    }
}
*/
