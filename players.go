package main

import (
    "github.com/gorilla/mux"
    "gopkg.in/mgo.v2/bson"
    "log"
    "net/http"
    "strconv"
    "encoding/json"
)

type Player struct {
    ID string `json:"id"`
    Score uint32 `json:"score"`
    TopScore uint32 `json:"topscore,omitempty"`
    Online bool `json:"online,omitempty"`
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
    json.NewEncoder(w).Encode(topPlayers)
    w.WriteHeader(200)
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

//func SaveScore(w http.ResponseWriter, r *http.Request) {
//    players := GetCollection("Players")
//
//    player, err := JsonToPlayer(r)
//    if err != nil {
//        log.Println(err)
//        w.WriteHeader(400)
//        return
//    }
//
//    if player.TopScore < player.Score {
//        player.TopScore = player.Score
//    }
//
//    var existingPlayer Player
//    err = players.Find(bson.M{"id":player.ID}).One(&existingPlayer)
//    if err != nil {
//        err = players.Insert(&player)
//        if err != nil {
//            log.Println(err)
//            w.WriteHeader(500)
//            return
//        }
//        w.WriteHeader(201)
//        return
//    }
//
//    fieldsToUpdate := bson.M{}
//    if existingPlayer.Score != player.Score {
//        fieldsToUpdate["score"] = player.Score
//    }
//    if existingPlayer.TopScore < player.Score {
//        fieldsToUpdate["topscore"] = player.Score
//    }
//
//    if len(fieldsToUpdate) != 0 {
//        err = players.Update(
//            bson.M{"id":player.ID},
//            bson.M{"$set": fieldsToUpdate},
//        )
//        if err != nil {
//            log.Println(err)
//            w.WriteHeader(500)
//            return
//        }
//
//        w.WriteHeader(200)
//    } else {
//        w.WriteHeader(304)
//    }
//}

func SaveScore(player *Player) {
    players := GetCollection("Players")

    if player.TopScore < player.Score {
        player.TopScore = player.Score
    }

    var existingPlayer Player
    err := players.Find(bson.M{"id":player.ID}).One(&existingPlayer)
    if err != nil {
        err = players.Insert(&player)
        if err != nil {
            log.Println(err)
            return
        }
        return
    }

    fieldsToUpdate := bson.M{}
    fieldsToUpdate["online"] = player.Online

    if existingPlayer.Score != player.Score {
        fieldsToUpdate["score"] = player.Score
    }
    if existingPlayer.TopScore < player.Score {
        fieldsToUpdate["topscore"] = player.Score
    }

    if len(fieldsToUpdate) != 0 {
        err = players.Update(
            bson.M{"id":player.ID},
            bson.M{"$set": fieldsToUpdate},
        )
        if err != nil {
            log.Println(err)
            return
        }

    }
}
