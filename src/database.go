package main

import (
    "github.com/globalsign/mgo"
)

var db *mgo.Database
var session *mgo.Session
var collections map[string]*mgo.Collection

func ConnectToDatabase() error {
    session, err := mgo.Dial("mongo:27017")
    //session, err := mgo.Dial("localhost")
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
