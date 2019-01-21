package main

import (
    "gopkg.in/mgo.v2/bson"
    "github.com/globalsign/mgo"
    "fmt"
)

var db *mgo.Database
var session *mgo.Session

func ConnectToDatabase() error {
    session, err := mgo.Dial("localhost")
    db = session.DB("JerkidoData")

    return err
}

func GetCollection(collectionName string) *mgo.Collection {
    return db.C(collectionName)
}

func DisconnectFromDatabase() {
    session.Close();
}

func PlayersInRoom(room *Room) ([]Player, []error) {
    var players []Player
    var errors  []error = nil
    var player Player

    for i := 0; i < len((*room).PlayerIDs); i++ {
        err := db.C("Players").Find(bson.M{"id":(*room).PlayerIDs[i]}).One(&player)
        if err != nil {
            fmt.Println("Error while looking for players in room. ")
            fmt.Println(err)
            errors = append(errors, err)
            continue
        }
        players = append(players, player)
    }

    return players, errors
}
