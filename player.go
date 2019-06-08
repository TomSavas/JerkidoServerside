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
    Score uint32 `json:"score,omitempty"`
    TopScore uint32 `json:"topscore,omitempty"`
    IsPlayer bool `json:"isplayer,omitempty"`
    Online bool `json:"online,omitempty"`
}

func NewPlayer(id string) *Player {
    return &Player{id, "", 0, 0, true, true}
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
    oldPlayer := &Player{player.ID, player.RoomID, player.Score, player.TopScore, player.IsPlayer, player.Online}

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
    playerInDB, err := GetExistingPlayer(player.ID)
    Log(err, "PlayerDB", "Player " + player.ID + " doesn't exist. Creating New.")

    if player.TopScore < player.Score {
        player.TopScore = player.Score
    }

    if player.TopScore < playerInDB.TopScore {
        player.TopScore = playerInDB.TopScore
    }

    _, err = GetPlayers().Upsert(bson.M{"id": player.ID}, bson.M{"$set": player})
    Fatal(err, "Failed saving player's score")
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

