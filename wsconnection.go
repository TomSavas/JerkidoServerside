package main

import (
    "net/http"
    "github.com/gorilla/websocket"
)

var DefaultUpgrader = websocket.Upgrader{}

type WSConnection struct {
    *websocket.Conn

    isConnectionClosed chan bool
}

func ToWSConnection(writer http.ResponseWriter, request *http.Request, upgrader websocket.Upgrader) (*WSConnection, error) {
    conn, err := upgrader.Upgrade(writer, request, nil)
    if err != nil {
        return nil, err
    }

    connection := &WSConnection{conn, make(chan bool, 1)}
    connection.setDefaultCloseHandler()

    return connection, nil
}

func (conn *WSConnection) IsClosed() bool {
    select {
        case value, _ := <-conn.isConnectionClosed:
            // The value was read (thus popped of) from isConnectionClosed channel if anyone else
            // tries to read the channel, the value will not be there. This writes the value that
            // has been read from the channel back to itself again.
            conn.isConnectionClosed <- value
            return true
        default:
            return false
    }
}

// Don't do any additional actions, besides writing to isConnectionClosed channel
func (conn *WSConnection) setDefaultCloseHandler() {
    conn.SetCloseHandler(func(code int, text string) error {
        return nil
    })
}

func (conn *WSConnection) SetCloseHandler(handler func(int, string) error) {
    conn.Conn.SetCloseHandler(func (code int, text string) error {
        err := handler(code, text)
        conn.isConnectionClosed <- true

        return err
    })
}

func (conn *WSConnection) Close() error {
    conn.isConnectionClosed <- true

    return conn.Conn.Close()
}
