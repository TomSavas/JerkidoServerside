package main

import (
    "net/http"
    "github.com/gorilla/websocket"
    "fmt"
)

func SetCloseHandler(connection *websocket.Conn, handler func(int, string) error) chan bool {
    connectionClosed := make(chan bool, 1)
    connection.SetCloseHandler(func (code int, text string) error {
        err := handler(code, text)

        connectionClosed <- true
        return err
    })

    return connectionClosed
}

func CheckForClosedConnection(connectionClosed chan bool, onClosedHandler func(interface{}) interface{}, onNotClosedHandler func() interface{}) interface{} {
    select {
        case value, _ := <- connectionClosed:
            return onClosedHandler(value)
        default:
            return onNotClosedHandler()
    }
}

func IsConnectionClosed(connectionClosed chan bool) bool {
    return CheckForClosedConnection(connectionClosed,
        func(chanValue interface{}) interface{} {
            //The value was read (thus popped of) from connectionClosed by
            //CheckForClosedConnection, if anyone else tries to read the channel,
            //the value will not be there. This writes the value that has been 
            //read from the channel back to itself again.
            connectionClosed <- chanValue.(bool)
            return true
        },
        func() interface{} {
            return false
        }).(bool)
}

func UpgradeConnToWebSocketConn(w http.ResponseWriter, r *http.Request) (*websocket.Conn, chan bool, error) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Print("Error upgrading http connection to a websocket: ")
        fmt.Println(err)
        return nil, nil, err
    }

    isConnectionClosed := SetCloseHandler(conn, func(code int, text string) error {
        fmt.Printf("User's connection closed.\n\tCode:%d \n\tText:%s", code, text)
        conn.Close()
        return nil
    })

    return conn, isConnectionClosed, err
}
