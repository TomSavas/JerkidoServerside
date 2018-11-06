package main

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "gopkg.in/mgo.v2/bson"
    "log"
    "net/http"
)

type Room struct {
    ID string `json:"id"`
    State RoomState `json:"state"`
    PlayerIDs []string `json:"playerids"`
}

type RoomInfo struct {
    BaseRoom Room `json:"room"`
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

func CreateRoom(w http.ResponseWriter, r *http.Request) {
    rooms := DB.C("Rooms");
    room := Room{GenerateUniqueRoomCode(), WaitingForPlayers, []string{}}

    err := rooms.Insert(&room)
    if err != nil {
        log.Println(err)
        w.WriteHeader(500)
        return
    }

    w.WriteHeader(201)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(room.ID)
}

func GenerateUniqueRoomCode() string {
    code := GenerateRoomCode(5);

    if(DB.C("Rooms").Find(bson.M{"ID":code}) == nil) {
        return GenerateUniqueRoomCode()
    }

    return code
}

func ConnectToRoom(w http.ResponseWriter, r *http.Request) {
    rooms := DB.C("Rooms")
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

    if !Exists(existingRoom.PlayerIDs, playerData["id"].(string)) {
        changedPlayerIDs := append(existingRoom.PlayerIDs, playerData["id"].(string))
        rooms.Update(
            bson.M{"id":existingRoom.ID},
            bson.M{"$set": bson.M{"playerids":changedPlayerIDs}},
        )
    }
}

func DisconnectFromRoom(w http.ResponseWriter, r *http.Request) {
    rooms := DB.C("Rooms")
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

func GetRoomInfo(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["roomID"]
    var room Room

    err := DB.C("Rooms").Find(bson.M{"id":id}).One(&room)
    if err != nil {
        log.Println(err)
        w.WriteHeader(410)
        return
    }

    roomInfo := RoomInfo{room, GatherPlayers(&room, DB)}

    w.WriteHeader(200)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(roomInfo)
}
