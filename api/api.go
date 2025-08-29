package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
  IdleTimer int `json:"idleTimer"`
}

type History struct {
  Type string `json:"type"`
  Songs []Response `jsons:"songs"`
}
var Songs []Response
var Status PlayerStatus
var TrackHistory History

func FetchCurrentSong(songChan chan Response) error {
  var pollInterval int
  if Status.IdleTimer >= 10000 {
    pollInterval = 20000
  } else {
    pollInterval = 6500
  }
  var previousSong Response
  var currentSong Response
  ticker := time.NewTicker(time.Duration(pollInterval) * time.Millisecond)
  
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
      fmt.Println("Error reading json: ", err)
      return err
    }

    if err := json.Unmarshal(respData, &currentSong); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      
    }
    //fmt.Println(song)
    currentSong.MetaData.AlbumArtURI = strings.Replace(
      currentSong.MetaData.AlbumArtURI, 
      "320x320.jpg", 
      "640x640.jpg", 1)

    if currentSong != previousSong && currentSong.MetaData.Album != "" {
      songChan <-currentSong
      Songs = append(Songs, currentSong)
      previousSong = currentSong
    }
  }
  return nil
}

func FetchCurrentStatus(statusChan chan PlayerStatus) error {
  pollInterval := 10000
  var currentStatus PlayerStatus
  var previousStatus PlayerStatus
  ticker := time.NewTicker(time.Duration(pollInterval) * time.Millisecond)

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
      fmt.Println("error getting from url:",err)
      return err
    }
    defer resp.Body.Close()

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
      fmt.Println("error reading:",err)
    }

    if err := json.Unmarshal([]byte(respData), &currentStatus); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      return err
    }
    if currentStatus != previousStatus {
      Status = currentStatus
      statusChan <- currentStatus
      previousStatus = currentStatus
    }
    if currentStatus.Status == "pause" {
      currentStatus.IdleTimer += 2000
    }
    if currentStatus.Status == "play" {
      currentStatus.IdleTimer = 0
    } 
  }
  return nil
}

func PlayerCommand(command string)error {
  tr := &http.Transport{
          TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
      } // 

  client := &http.Client{
    Timeout: 5 * time.Second,
    Transport: tr,
  }
  var url string 
  switch command {
    case "play":
    url = "https://10.0.0.119/httpapi.asp?command=setPlayerCmd:play"
  case "pause":
    url = "https://10.0.0.119/httpapi.asp?command=setPlayerCmd:pause"
  case "onepause":
    url = "https://10.0.0.119/httpapi.asp?command=setPlayerCmd:onepause" // toggle pause/pause
  case "next":
    url = "https://10.0.0.119/httpapi.asp?command=setPlayerCmd:next"
  case "previous":
    url = "https://10.0.0.119/httpapi.asp?command=setPlayerCmd:next"
  }

  resp, err := client.Get(url)
  if err != nil {
    return err
  }
  defer resp.Body.Close()
  
  return nil
}



