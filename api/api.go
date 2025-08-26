package api

import (
  "fmt"
  "encoding/json"
  "crypto/tls"
  "net/http"
  "log"
  "time"
  "io"
  "strings"
)


type Response struct {
  MetaData struct {
    Album string `json:"album"`
    Title string `json:"title"`
    Artist string `json:"artist"`
    AlbumArtURI string `json:"albumArtURI"`
    SampleRate string `json:"sampleRate"`
    BitDepth string `json:"bitDepth"`
  } `json:"metaData"`
} 

type PlayerStatus struct {
  Status string `json:"status"`
}


func SendCurrentSong(songChan chan Response) error {
  var previousSong Response
  var currentSong Response
  ticker := time.NewTicker(5000 * time.Millisecond)
  
  for range ticker.C {
    tr := &http.Transport{
          TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
      } // bypass tls
    url := "https://10.0.0.119/httpapi.asp?command=getMetaInfo"

    client := &http.Client{
      Timeout: 5 * time.Second,
      Transport: tr,
    }
    resp, err := client.Get(url)

    if err != nil {
      fmt.Print(err.Error())
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
      log.Fatal(err)
    }

    if err := json.Unmarshal(respData, &currentSong); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      return err
    }
    //fmt.Println(song)
    currentSong.MetaData.AlbumArtURI = strings.Replace(
      currentSong.MetaData.AlbumArtURI, 
      "320x320.jpg", 
      "640x640.jpg", 1)

    if currentSong != previousSong {
      songChan <-currentSong
      previousSong = currentSong
    }
  }
  return nil
}

func FetchCurrentStatus(statusChan chan PlayerStatus) error {
  var currentStatus PlayerStatus
  var previousStatus PlayerStatus
  ticker := time.NewTicker(5000 * time.Millisecond)

  for range ticker.C {
    tr := &http.Transport{
          TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
      } // bypass tls
    
    url := "https://10.0.0.119/httpapi.asp?command=getPlayerStatus"
    client := &http.Client{
      Timeout: 5 * time.Second,
      Transport: tr,
    }
    resp, err := client.Get(url)

    if err != nil {
      log.Fatal(err)
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
      log.Fatal(err)
    }

    if err := json.Unmarshal([]byte(respData), &currentStatus); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      return err
    }
    if currentStatus != previousStatus {
      statusChan <- currentStatus
      previousStatus = currentStatus
    }
  }
  return nil
}
