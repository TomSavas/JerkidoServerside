package main

import (
    "net/http"
    "github.com/gorilla/websocket"
    "time"
)

type WSConnection struct {
    *websocket.Conn

    isConnectionClosed chan bool
}

func ToWSConnection(writer http.ResponseWriter, request *http.Request, upgrader *websocket.Upgrader) *WSConnection {
    if upgrader == nil {
        upgrader = &websocket.Upgrader{}

        upgrader.CheckOrigin = func(r * http.Request) bool {
            return true
        }
    }

    conn, err := upgrader.Upgrade(writer, request, nil)
    Fatal(err, "Failed to upgrade HTTP connection to a WS connection")

    connection := &WSConnection{conn, make(chan bool, 1)}
    connection.setDefaultCloseHandler()

    return connection
}

func (conn *WSConnection) IsClosed() bool {
    select {
        case value, _ := <- conn.isConnectionClosed:
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

    // Send ws close control message
    conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Millisecond))
    time.Sleep(10 * time.Millisecond)

    return conn.Conn.Close()
}
