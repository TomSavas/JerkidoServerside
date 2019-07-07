package main

import (
    "io/ioutil"
    "encoding/json"
    "net/http"
    //"github.com/gorilla/websocket"
    "os"
    "os/exec"
    "fmt"
    "time"
)

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

func WsJsonToMap(bytes []byte) (map[string]interface{}, error) {
    parsedJson := make(map[string]interface{})

    err := json.Unmarshal(bytes, &parsedJson)

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

func ChannelHasValue(channel chan interface{}) (bool, interface{}) {
    select {
        case value, _ := <- channel:
            return true, value
        default:
            return false, nil
    }
}

func ChannelHasValueWithTimeout(channel chan interface{}, timeoutInSeconds int) (bool, interface{}) {
    select {
        case value, _ := <- channel:
            return true, value
        case <- time.After(time.Duration(timeoutInSeconds) * time.Second):
            return false, nil
    }
}

func DisconnectPlayer(player *Player, room *Room, connection *WSConnection) {
    player.Online = false
    player.SaveScore()
    room.RemovePlayer(player.ID)
    connection.Close()
}

func ClearTerminal() {
    c := exec.Command("clear")
    c.Stdout = os.Stdout
    c.Run()
}

func LogInfo(tag string, msg string) {
    fmt.Printf("[%s] <%s>: %s\n", time.Now().UTC().String(), tag, msg)
}
