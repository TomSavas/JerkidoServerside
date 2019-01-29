package main

import (
    "gopkg.in/mgo.v2/bson"
    "github.com/globalsign/mgo"
    "fmt"
)

var db *mgo.Database
var session *mgo.Session
var collections map[string]*mgo.Collection

func ConnectToDatabase() error {
    session, err := mgo.Dial("localhost")
    db = session.DB("JerkidoData")
    collections = make(map[string]*mgo.Collection)

    return err
}

func GetCollection(collectionName string) *mgo.Collection {
    if collection, exists := collections[collectionName]; exists {
        return collection
    }
    collections[collectionName] = db.C(collectionName);

    return collections[collectionName];
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
