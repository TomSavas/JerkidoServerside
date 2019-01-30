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

type RoomState int
const (
    WaitingForPlayers RoomState = 0
    Transition        RoomState = 1
    CountingDown_3    RoomState = 2
    CountingDown_2    RoomState = 3
    CountingDown_1    RoomState = 4
    CountingDown_0    RoomState = 5
    Play              RoomState = 6
    End               RoomState = 7
)

type Room struct {
    ID           string    `json:"id"`
    State        RoomState `json:"state"`
    PlayerIDs    []string  `json:"playerids"`
    ObserverIDs []string  `json:"observerids"`
    OwnerID      string    `json:"ownerid"`
}

type RoomInfo struct {
    ID      string    `json:"id"`
    State   RoomState `json:"state"`
    Players []Player  `json:"players"`
}

func NewRoom(ownerID string, creatorIsPlayer bool) *Room {
    var players []string
    var observers []string

    if creatorIsPlayer {
        players = []string{ownerID}
    } else {
        observers = []string{ownerID}
    }

    return &Room{GenerateUniqueRoomCode(GetRooms()), WaitingForPlayers, players, observers, ownerID}
}

func GetRoom(roomID string) (*Room, error) {
    room := new(Room)
    room.ID = roomID
    return room, room.Update()
}

// Returns a value representing whether the room was upated
func (room *Room) UpdateWithStatusReport() (bool, error) {
    oldRoom := &Room{room.ID, room.State, room.PlayerIDs, room.ObserverIDs, room.OwnerID}

    err := room.Update()
    if err != nil {
        return false, err
    }

    return !room.Equals(oldRoom), nil
}

func (room *Room) Update() error {
    return GetRooms().Find(bson.M{"id": room.ID}).One(room)
}

func (room *Room) ChangeRoomStateWithoutUpdating(state RoomState) error {
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

func (room *Room) ChangeRoomState(state RoomState) error {
    room.State = state

    return room.ChangeRoomStateWithoutUpdating(state)
}

func (room *Room) Save() error {
    return GetRooms().Insert(room)
}

// Returns a struct that's meant to be sent to the user
func (room *Room) Info() (*RoomInfo, []error) {
    players, errors := PlayersInRoom(room)
    return &RoomInfo{room.ID, room.State, players}, errors
}

func (room *Room) Equals(otherRoom *Room) bool {
    if room.ID == otherRoom.ID && room.State == otherRoom.State && room.OwnerID == otherRoom.OwnerID && len(room.PlayerIDs) == len(otherRoom.PlayerIDs) && len(room.ObserverIDs) == len(otherRoom.ObserverIDs) {
        equals := func(firstList []string, secondList []string) bool {
            sort.Strings(firstList)
            sort.Strings(secondList)

            for i := 0; i < len(firstList); i++ {
                if (firstList[i] != secondList[i]) {
                    return false
                }
            }
            return true
        }

        return equals(room.PlayerIDs, otherRoom.PlayerIDs) && equals(room.ObserverIDs, otherRoom.ObserverIDs)
    } else {
        return false
    }
}

func GetRooms() *mgo.Collection {
    return GetCollection("Rooms")
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
    connection, err := ToWSConnection(w, r, nil)

    var owner User
    err = connection.ReadJSON(&owner)
    if err != nil {
        fmt.Print("Error reading room creator. ")
        fmt.Println(err)
        return
    }

    //Overides highscore
    SaveScore(&Player{owner.ID, 0, 0, true})
    room := NewRoom(owner.ID, owner.IsPlayer)
    fmt.Println("Created a new room: " + room.ID + " owner: " + room.OwnerID)

    err = room.Save()
    if err != nil {
        fmt.Println(err)
        return
    }
    connection.WriteMessage(websocket.TextMessage, []byte("{\"id\":\""+room.ID+"\"}"))

    if owner.IsPlayer {
        ControlRoom(w, r, connection, room, JoinRoom)
    } else {
        ControlRoom(w, r, connection, room, ObserveRoom)
    }
}

func Observe(w http.ResponseWriter, r *http.Request) {
    //connection, isConnectionClosed, err := UpgradeConnToWebSocketConn(w, r)
    connection, err := ToWSConnection(w, r, nil)

    var observer User
    err = connection.ReadJSON(&observer)
    if err != nil {
        fmt.Printf("Error reading observer: %+v: ", observer)
        fmt.Println(err)
        return
    }
    fmt.Println("Observer connected: " + observer.ID)
    room, err := GetRoom(observer.RoomID)

    if err != nil {
        fmt.Printf("Room doesnt exist: %+v: ", observer)
        fmt.Println(err)
        return
    }

    ObserveRoom(w, r, connection, room)
}

func ObserveRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room) {
    fmt.Println("Room: " + room.ID + " owner: " + room.OwnerID)

    //If there is no Read method active, the closeHandler is not triggered...
    go func() {
        for {
            if connection.IsClosed() {
                fmt.Println("Connection closed by the observer")
                return
            }

            _, _, _ = connection.ReadMessage()
        }
    }()

    maxOnline := 0
    for {
        someOnline := 0
        if connection.IsClosed() {
            fmt.Println("Connection closed by the observer (in observe)")
            return
        }

        err := room.Update()

        if err != nil {
            fmt.Print("Failed to read rooms: ")
            fmt.Println(err)
            break
        }

        roomInfo, errs := room.Info()

        for _, err = range errs {
            if err != nil {
                fmt.Println(err)
                //return
            }
        }

        for _, player := range roomInfo.Players {
            if player.Online {
                someOnline++
            }
        }

        if someOnline > maxOnline {
            maxOnline = someOnline
        }

        /*
         * Output logic
         */
        ClearTerminal()
        fmt.Printf("People online: %d\n", someOnline)
        fmt.Printf("Max people online: %d\n", maxOnline)
        switch room.State {
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
            sort.Slice(roomInfo.Players, func(i, j int) bool { return roomInfo.Players[i].Score > roomInfo.Players[j].Score })

            fmt.Println("Room: " + room.ID + " owner: " + room.OwnerID)
            for _, player := range roomInfo.Players {
                fmt.Printf("ID: %s, score: %d, online: %t\n", player.ID, player.Score, player.Online)
            }
        case End:
            fmt.Println("Game ended")
        default:
            fmt.Println("Wrong game state")
        }
        /*
         * Output logic
         */

        err = connection.WriteJSON(*roomInfo)
        if err != nil {
            if connection.IsClosed() {
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

func ControlRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room, onPlay func(http.ResponseWriter, *http.Request, *WSConnection, *Room)) {
    // At this point the room must have the state of WaitingForPlayers
    roomStateChanged := make(chan interface{}, 1)
    go WaitForRoomStateUpdate(connection, roomStateChanged)

	for {
        if connection.IsClosed() {
            fmt.Println("Connection closed by the observer")
            return
        }

        roomUpdated, err := room.UpdateWithStatusReport()
		if err != nil {
			fmt.Print("Failed to read rooms: ")
			fmt.Println(err)
			break
		}

		roomInfo, errs := room.Info()
		for _, err = range errs {
			if err != nil {
				fmt.Println("Error while parsing room to RoomInfo")
				fmt.Println(err)
				return
			}
		}

		if hasValue, _ := ChannelHasValue(roomStateChanged); hasValue {
            go func() {
                room.ChangeRoomStateWithoutUpdating(Transition)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomStateWithoutUpdating(CountingDown_3)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomStateWithoutUpdating(CountingDown_2)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomStateWithoutUpdating(CountingDown_1)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomStateWithoutUpdating(CountingDown_0)
                time.Sleep(1000 * time.Millisecond)
                room.ChangeRoomStateWithoutUpdating(Play)
		    }()
            onPlay(w, r, connection, room)
            break
		}

        if roomUpdated {
            ClearTerminal()
            for _, player := range roomInfo.Players {
                fmt.Printf("ID: %s, score: %d, online: %t\n", player.ID, player.Score, player.Online)
            }

            err = connection.WriteJSON(*roomInfo)
            if err != nil {
                if connection.IsClosed() {
                    fmt.Println("Connection closed by the controling user (in control)")
                    return
                } else {
                    fmt.Println("Cannot write room info to the controling user")
                    fmt.Println(err)
                }
            }
        }

		time.Sleep(100 * time.Millisecond)
	}

}

func WaitForRoomStateUpdate(connection *WSConnection, roomStateChanged chan interface{}) {
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
	//connection, isConnectionClosed, err := UpgradeConnToWebSocketConn(w, r)
    connection, err := ToWSConnection(w, r, nil)

	var user User
	err = connection.ReadJSON(&user)
	if err != nil {
		fmt.Print("Error reading connecting user: ")
		fmt.Println(err)
		return
	}

	fmt.Println("looking for a room")
    room, err := GetRoom(user.RoomID)

	if err != nil {
		fmt.Println("Failed geting room with id: " + user.RoomID)
		fmt.Println(err)
		return
	}

	if !Exists(room.PlayerIDs, user.ID) {
		SaveScore(&Player{user.ID, 0, 0, true}) //Don't leave this. It will override an existing player's top score.

		changedPlayerIDs := append(room.PlayerIDs, user.ID)
		err = GetRooms().Update(
			bson.M{"id": room.ID},
			bson.M{"$set": bson.M{"playerids": changedPlayerIDs}},
		)
		if err != nil {
			fmt.Println("Failed adding player to playerids in room: " + user.RoomID)
			fmt.Println(err)
		}
	}

	PlayGame(w, r, connection, room)
}

func JoinRoom(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room) {
    //err := rooms.Find(bson.M{"id": roomID}).One(&existingRoom)

	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	//if !Exists(existingRoom.PlayerIDs, existingRoom.OwnerID) {
	//	changedPlayerIDs := append(existingRoom.PlayerIDs, existingRoom.OwnerID)
	//	rooms.Update(
	//		bson.M{"id": existingRoom.ID},
	//		bson.M{"$set": bson.M{"playerids": changedPlayerIDs}},
	//	)
	//}

	PlayGame(w, r, connection, room)
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *WSConnection, room *Room) {
    //fmt.Println("playing...")
	var player Player
	player.Online = true

	//var room Room
    //err := GetRooms().Find(bson.M{"id": roomID}).One(&room)
    //players, _ := PlayersInRoom(&room)
    roomInfo, _ := room.Info()
	connection.WriteJSON(&roomInfo)

	previousRoomState := WaitingForPlayers
	for {
		if connection.IsClosed() {
			fmt.Println("Player disconnected")
			player.Online = false
			SaveScore(&player)
			return
		}

        err := room.Update()

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
			if connection.IsClosed() {
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
        err = room.Update()

        roomInfo, _ := room.Info()
        sort.Slice(roomInfo.Players, func(i, j int) bool { return roomInfo.Players[i].Score > roomInfo.Players[j].Score })

        fmt.Println("Room: " + room.ID + " owner: " + room.OwnerID)
        for _, player := range roomInfo.Players {
            fmt.Printf("ID: %s, score: %d, online: %t\n", player.ID, player.Score, player.Online)
        }
        // for debugging purposes
	}
}
