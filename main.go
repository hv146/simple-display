package main

import (
  "fmt"
  "net/http"
  "github.com/gorilla/websocket"
  "wss-test/api"
  "encoding/json"
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

    api.PlayerCommand(string(msg))
    handleHistory(string(msg))
  }
}

func main() {
  songChan := make(chan api.Response)
  statusChan := make(chan api.PlayerStatus)

  go api.FetchCurrentSong(songChan)
  go api.FetchCurrentStatus(statusChan)

  go handleBroadcasting(songChan, statusChan)

  http.Handle("/", http.FileServer(http.Dir("./static/")))
  http.HandleFunc("/ws", handleConnections)

  fmt.Println("WebSocket server started on: 8080")
  err := http.ListenAndServe(":8080", nil)
  if err != nil {
    fmt.Println("ListenAndServe:", err)
  }
}

func handleBroadcasting(songChan chan api.Response, statusChan chan api.PlayerStatus) {
  for {
    select {
    case song := <-songChan:
      jsonSong, _ := json.Marshal(song)
      for client := range clients {
          if err := client.WriteMessage(websocket.TextMessage, jsonSong); err != nil {
            fmt.Println("broadcast error:", err)
            client.Close()
            delete(clients, client)
          }
      }
    case status := <-statusChan:
      jsonStatus, _ := json.Marshal(status)
      for client := range clients {
          if err := client.WriteMessage(websocket.TextMessage, jsonStatus); err != nil {
            fmt.Println("broadcast error:", err)
            client.Close()
            delete(clients, client)
          }
        }
    }
  }
}


func handleHistory(msg string) {
  api.TrackHistory.Type = "history"
  api.TrackHistory.Songs = api.Songs
  if (msg == "request"){
    jsonHistory,_ := json.Marshal(api.TrackHistory)
    for client := range clients {
      if err := client.WriteMessage(websocket.TextMessage, jsonHistory); err != nil {
              fmt.Println("broadcast error:", err)
              client.Close()
              delete(clients, client)
      }
    }
  }
}


