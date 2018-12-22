package main

import (
    //"encoding/json"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "gopkg.in/mgo.v2/bson"
    "log"
    "net/http"
    "math/rand"
    "time"
    "bytes"
    "github.com/globalsign/mgo"
    "fmt"
    "os"
    "os/exec"
    "sort"
)

type Room struct {
    ID string `json:"id"`
    State RoomState `json:"state"`
    PlayerIDs []string `json:"playerids"`
    ObeserverIDs []string `json:"observerids"`
    OwnerID string `json:"ownerid"`
}

type RoomInfo struct {
    ID string `json:"id"`
    State RoomState `json:"state"`
    Players []Player `json:"players"`
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
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Error upgrading http connection to a websocket.")
        return
    }
    defer conn.Close()

    var owner User
    err = conn.ReadJSON(&owner)
    if err != nil {
        fmt.Println("Error reading message")
        return
    }

    rooms := GetRooms()
    room := Room{GenerateUniqueRoomCode(rooms), Play, []string{}, []string{owner.ID}, owner.ID}

    fmt.Println("Room: " + room.ID + " owner: " + room.OwnerID)

    err = rooms.Insert(&room)
    if err != nil {
        log.Println(err)
        return
    }
    conn.WriteMessage(websocket.TextMessage, []byte("{\"id\":\"" + room.ID + "\"}"))

    if owner.IsPlayer {
        //JoinRoom(w, r, connection, room.ID)
    } else {
        ObserveRoom(w, r, conn, room.ID)
    }

    //ShowProgress(w, r, conn, room.ID)
}

func Observe(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Error upgrading http connection to a websocket.")
        return
    }
    defer conn.Close()

    var observer User
    err = conn.ReadJSON(&observer)
    if err != nil {
        fmt.Printf("Error reading observer: %+v", observer)
        return
    }
    fmt.Println("Observer connected: " + observer.ID)

    ObserveRoom(w, r, conn, observer.RoomID)
}

func ObserveRoom(w http.ResponseWriter, r *http.Request, connection *websocket.Conn, roomID string) {
    clear := func() {
        c := exec.Command("clear")
        c.Stdout = os.Stdout
        c.Run()
    }

    connectionClosed := make(chan bool, 1)

    connection.SetCloseHandler(func (code int, text string) error {
        fmt.Printf("Observer's connection closed.\n\tCode:%d \n\tText:%s", code, text)

        connectionClosed <- true
        return nil
    })

    defer connection.Close()
    var existingRoom Room
    rooms := GetRooms()
    err := rooms.Find(bson.M{"id":roomID}).One(&existingRoom)
    fmt.Println("Room: " + existingRoom.ID + " owner: " + existingRoom.OwnerID)

    someOnline := 0
    maxOnline := 0
    updatedAfterLastDisconnect := false
    for {
        select {
            case _, _ = <- connectionClosed:
                fmt.Println("Connection broken")
                return
            default:
                //fmt.Println("Running")
                someOnline = 0
                //continue
        }

        err = rooms.Find(bson.M{"id":roomID}).One(&existingRoom)

        if err != nil {
            log.Println("Failed to read rooms")
            break
        }

        players, errs := PlayersInRoom(&existingRoom)

        for _, err = range(errs) {
            if err != nil {
                log.Println(err)
                //return
            }
        }

        for _, player := range(players) {
            if player.Online {
                someOnline++
                updatedAfterLastDisconnect = false
            }
        }

        if someOnline > maxOnline {
            maxOnline = someOnline
        }

        if someOnline < 1 {
            if !updatedAfterLastDisconnect {
                updatedAfterLastDisconnect = true
            } else {
                continue
            }
        }

        clear()
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
                for _, player := range(players) {
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
            log.Println(err)
        }

        time.Sleep(100 * time.Millisecond)
    }
}

func GenerateUniqueRoomCode(rooms *mgo.Collection) string {
    code := GenerateRoomCode(5);

    if(rooms.Find(bson.M{"ID":code}) == nil) {
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

func JoinRoom(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

    var user User
    err := conn.ReadJSON(&user)
    if err != nil {
        fmt.Println("Error reading connecting user")
        return
    }

    rooms := GetRooms()
    //playerData, err := WsJsonToMap(msg)
    if err != nil {
        log.Println(err)
        return
    }

    var existingRoom Room
    err = rooms.Find(bson.M{"id":user.RoomID}).One(&existingRoom)

    if err != nil {
        log.Println(err)
        return
    }

    if !Exists(existingRoom.PlayerIDs, user.ID) {
        changedPlayerIDs := append(existingRoom.PlayerIDs, user.ID)
        rooms.Update(
            bson.M{"id":existingRoom.ID},
            bson.M{"$set": bson.M{"playerids":changedPlayerIDs}},
        )
    }

    PlayGame(w, r, conn)
}

func PlayGame(w http.ResponseWriter, r *http.Request, connection *websocket.Conn) {
    defer connection.Close()

    var player Player
    player.Online = true
    connectionClosed := make(chan bool, 1)

    connection.SetCloseHandler(func (code int, text string) error {
        log.Printf("Player (ID: %s, score: %d) connection closed.\n\tCode:%d \n\tText:%s",player.ID, player.Score, code, text)
        connectionClosed <- true

        return nil
    })

    for {
        select {
            case _, _ = <- connectionClosed:
                fmt.Println("Player disconnected")
                player.Online = false
                SaveScore(&player)
                return
            default:
                err := connection.ReadJSON(&player)
                if err != nil {
                    fmt.Println("Error reading message")
                    break
                } else {
                    SaveScore(&player)
                }
        }
    }
}

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
