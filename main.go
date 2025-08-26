package main

import (
  "fmt"
  "net/http"
  "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {return true},
}

var clients = make(map[*websocket.Conn]bool) //track active clients
func handleConnections(w http.ResponseWriter, r *http.Request) {

  ws, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    fmt.Println(err)
    return
  }
  defer ws.Close()
  clients[ws] = true
  for {
    _, msg, err := ws.ReadMessage()
    if err != nil {
      fmt.Println("read error:", err)
      delete(clients, ws)
      break
    }
    fmt.Printf("Received: %s\n", msg)
    
    for client := range clients {
      if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil{
        fmt.Println("broadcast error:", err)
        client.Close()
        delete(clients, client)
      }
    }
  }
}

func main() {
  http.HandleFunc("/ws", handleConnections)

  fmt.Println("WebSocket server started on: 8080")
  err := http.ListenAndServe(":8080", nil)
  if err != nil {
    fmt.Println("ListenAndServe:", err)
  }
}
