package main

import (
    "io/ioutil"
    "encoding/json"
    "net/http"
    "time"
    "math/rand"
    "bytes"
    "github.com/globalsign/mgo"
    "gopkg.in/mgo.v2/bson"
    "log"
)

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

func GatherPlayers(room *Room, db *mgo.Database) []Player {
    var players []Player
    var player Player

    for i := 0; i < len((*room).PlayerIDs); i++ {
        err := db.C("Players").Find(bson.M{"id":(*room).PlayerIDs[i]}).One(&player)
        if err != nil {
            log.Println(err)
            continue
        }
        players = append(players, player)
    }

    return players
}

func Exists(items []string, item string) bool {
    for i := 0; i < len(items); i++ {
        if item == items[i] {
            return true
        }
    }
    return false
}

func JsonToMap(r *http.Request) (map[string]interface{}, error) {
    body, err := ioutil.ReadAll(r.Body)
    parsedJson := make(map[string]interface{})

    if err != nil {
        return parsedJson, err
    }
    err = json.Unmarshal(body, &parsedJson)

    return parsedJson, err
}

func JsonToPlayer(r *http.Request) (Player, error) {
    body, err := ioutil.ReadAll(r.Body)
    var player Player

    if err != nil {
        return player, err
    }
    err = json.Unmarshal(body, &player)

    if player.TopScore == 0 {
        player.TopScore = player.Score
    }

    return player, err
}
