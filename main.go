package main

import (
  "fmt"
  "net/http"
  "github.com/gorilla/websocket"
  "wss-test/api"
  "encoding/json"
  "database/sql"
  "log"
  _ "github.com/mattn/go-sqlite3"
)

var upgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {return true},
}

var clients = make(map[*websocket.Conn]bool) //track active clients
var songChan chan api.Response
var statusChan chan api.PlayerStatus

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
  //createSongDB()
  songChan = make(chan api.Response)
  statusChan = make(chan api.PlayerStatus)

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
      updateSongDB(song)
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
  history := getSongDB()
  history.Type = "history"
  if (msg == "requestHistory"){
    jsonHistory,_ := json.Marshal(history)
    for client := range clients {
      if err := client.WriteMessage(websocket.TextMessage, jsonHistory); err != nil {
              fmt.Println("broadcast error:", err)
              client.Close()
              delete(clients, client)
      }
    }
  }
}


func createSongDB() {
  db, err := sql.Open("sqlite3", "songs.db")
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()
  sqlStmt := `
  CREATE TABLE IF NOT EXISTS songs(
      album TEXT,
      title TEXT,
      artist TEXT,
      albumArtURI TEXT,
      sampleRate TEXT,
      bitDepth TEXT
  )
  `
  _, err = db.Exec(sqlStmt)
  if err != nil {
    log.Fatal(err)
  }
  log.Println("SongDB created successfully")
}

func updateSongDB(song api.Response) {
  query := "INSERT INTO songs(album, title, artist, albumArtURI, sampleRate, bitDepth) VALUES (?, ?, ?, ?, ?, ?)"

  data := song.MetaData
  db, err := sql.Open("sqlite3", "./songs.db")
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  _, err = db.Exec(query, data.Album, data.Title, data.Artist, data.AlbumArtURI, data.SampleRate, data.BitDepth)
  if err != nil {
    log.Fatal(err)
  }
  log.Println("Update DB successful")
}

func getSongDB() api.History {//return DB
  history := api.History{}
  songs := []api.Response{}

  db, err := sql.Open("sqlite3", "./songs.db")
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  rows, err := db.Query("SELECT album, title, artist, albumArtURI, sampleRate, bitDepth FROM songs")
  if err != nil {
    log.Fatal(err)
  }

  defer rows.Close()

  for rows.Next() {
    var album string
    var title string
    var artist string
    var albumArtURI string
    var sampleRate string
    var bitDepth string
    err = rows.Scan(&album, &title, &artist, &albumArtURI, &sampleRate, &bitDepth)
    
    song := api.Response{}
    song.MetaData.Album = album
    song.MetaData.Title = title
    song.MetaData.Artist = artist
    song.MetaData.AlbumArtURI = albumArtURI
    song.MetaData.SampleRate = sampleRate
    song.MetaData.BitDepth = bitDepth
    
    if album != "" {
      songs = append(songs, song)
    } else {
      continue
    }
  }
  if err = rows.Err(); err != nil {
    log.Fatal(err)
  }
  history.Songs = songs
  return history
}


