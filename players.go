package main

import (
    "github.com/gorilla/mux"
    "gopkg.in/mgo.v2/bson"
    "log"
    "net/http"
    "strconv"
    "encoding/json"
    "github.com/globalsign/mgo"
)

type Player struct {
    ID string `json:"id"`
    RoomID string `json:"roomid,omitempty"`
    IsPlayer bool `json:"isplayer,omitempty"`
    Online bool `json:"online,omitempty"`
    Score uint32 `json:"score,omitempty"`
    TopScore uint32 `json:"topscore,omitempty"`
}

func NewPlayer(id string) *Player {
    return &Player{id, "", true, true, 0, 0}
}

func GetPlayers() *mgo.Collection {
    return GetCollection("Players")
}

func GetExistingPlayer(id string) (*Player, error) {
    player := new(Player)
    err := GetPlayers().Find(bson.M{"id":id}).One(player)

    return player, err
}

// Returns a value representing whether the room was upated
func (player *Player) UpdateWithStatusReport() (bool, error) {
    oldPlayer := &Player{player.ID, player.RoomID, player.IsPlayer, player.Online, player.Score, player.TopScore}

    err := player.Update()
    if err != nil {
        return false, err
    }

    return *player == *oldPlayer, nil
}

func (player *Player) Update() error {
    return GetPlayers().Find(bson.M{"id": player.ID}).One(player)
}

func (player *Player) SaveScore() {
    if player.TopScore < player.Score {
        player.TopScore = player.Score
    }

    playerInDB, err := GetExistingPlayer(player.ID)
    if err != nil {
        err = GetPlayers().Insert(player)
        if err != nil {
            log.Println(err)
            return
        }
        return
    }

    fieldsToUpdate := bson.M{}
    fieldsToUpdate["online"] = player.Online
    if playerInDB.Score != player.Score {
        fieldsToUpdate["score"] = player.Score
    }
    if playerInDB.TopScore < player.Score {
        fieldsToUpdate["topscore"] = player.Score
    }

    if len(fieldsToUpdate) != 0 {
        err = GetPlayers().Update(
            bson.M{"id":player.ID},
            bson.M{"$set": fieldsToUpdate},
        )
        if err != nil {
            log.Println(err)
            return
        }
    }
}

func GetTopPlayers(w http.ResponseWriter, r *http.Request) {
    players := GetCollection("Players")
    amountOfTopPlayers, err := strconv.Atoi(mux.Vars(r)["amountOfTopPlayers"])
    if err != nil {
        log.Println(err)
        w.WriteHeader(400)
        return
    }

    var topPlayers []Player
    err = players.Find(nil).Sort("-topscore").Limit(amountOfTopPlayers).All(&topPlayers)
    if err != nil {
        log.Println(err)
        w.WriteHeader(500)
        return
    }
    w.WriteHeader(200)
    json.NewEncoder(w).Encode(topPlayers)
}

func GetPosition(w http.ResponseWriter, r *http.Request) { 
    playerID := mux.Vars(r)["playerID"]
    players := GetCollection("Players")
    var topPlayers []Player

    err := players.Find(nil).Sort("-topscore").All(&topPlayers)
    if err != nil {
        log.Println(err)
        w.WriteHeader(500)
        return
    }

    position := 0
    for i := 0; i < len(topPlayers); i++ {
        if topPlayers[i].ID == playerID {
            position = i
            break;
        }
    }
    position += 1

    json.NewEncoder(w).Encode(map[string]int{"position": position})
    w.WriteHeader(200)
}

