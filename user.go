package main

import (
    "io/ioutil"
    "encoding/json"
    "net/http"
)

type User struct {
    ID string `json:"id"`
    RoomID string `json:"roomid,omitempty"`
    IsPlayer bool `json:"isplayer,omitempty"`
}

func GetUserFromRequest(request *http.Request) (User, error) {
    body, err := ioutil.ReadAll(request.Body)
    var user User

    if err != nil {
        return user, err
    }
    err = json.Unmarshal(body, &user)

    return user, err
}
