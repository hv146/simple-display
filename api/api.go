package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
  "strconv"
  "sync"
)
var (
	httpClient *http.Client
	clientOnce sync.Once
)

func getHTTPClient() *http.Client {
	clientOnce.Do(func() {
		tr := &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        2,                // Limit total idle connections
			MaxIdleConnsPerHost: 1,                // Only 1 connection to WiiM at a time
			IdleConnTimeout:     60 * time.Second, // Keep connections alive longer
			DisableKeepAlives:   false,            // Enable keep-alive
		}
		
		httpClient = &http.Client{
			Timeout:   15 * time.Second, // Reduced timeout
			Transport: tr,
		}
	})
	return httpClient
}

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
  Totlen string `json:"totlen"`
  Curpos string  `json:"curpos"`
}

type History struct {
  Type string `json:"type"`
  Songs []Response `jsons:"songs"`
}
var Songs []Response
var Status PlayerStatus
var TrackHistory History

var lastAPICall time.Time
var apiMutex sync.Mutex

func rateLimitedRequest(url string) (*http.Response, error) {
	apiMutex.Lock()
	defer apiMutex.Unlock()
	
	// Ensure minimum 500ms between any API calls
	if time.Since(lastAPICall) < 500*time.Millisecond {
		time.Sleep(500*time.Millisecond - time.Since(lastAPICall))
	}
	
	client := getHTTPClient()
	resp, err := client.Get(url)
	lastAPICall = time.Now()
	
	return resp, err
}

func FetchCurrentSong(songChan chan Response) error {
  var previousSong Response
  var currentSong Response

  currentSong.MetaData.Album = "unknow"
  currentSong.MetaData.Title = "unknow"
  currentSong.MetaData.Artist = "unknow"
  currentSong.MetaData.AlbumArtURI = "unknow"
  currentSong.MetaData.BitDepth = "unknow"
  currentSong.MetaData.SampleRate = "unknow"
  
  basePollInterval := 8 * time.Second  // Increased base interval
	slowPollInterval := 30 * time.Second 

  ticker := time.NewTicker(basePollInterval)
  defer ticker.Stop()

  for range ticker.C {
    
    var pollInterval time.Duration
		if Status.Status == "stop" || Status.IdleTimer >= 10000 {
			pollInterval = slowPollInterval
		} else {
			pollInterval = basePollInterval
		}
		
		// Reset ticker if interval changed
		ticker.Reset(pollInterval)
		
		url := "https://10.0.0.120/httpapi.asp?command=getMetaInfo"
		resp, err := rateLimitedRequest(url)

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
      fmt.Println("Error reading json: ", err)
      continue
    }
    if err := json.Unmarshal(respData, &currentSong); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      
    }
    resp.Body.Close()
    //fmt.Println(song)
    currentSong.MetaData.AlbumArtURI = strings.Replace(
      currentSong.MetaData.AlbumArtURI, 
      "320x320.jpg", 
      "640x640.jpg", 1)
    currentSong.MetaData.AlbumArtURI = strings.Replace(
      currentSong.MetaData.AlbumArtURI, 
      "https", 
      "http", 1)


    if currentSong != previousSong && currentSong.MetaData.Album != "unknow" {
      songChan <-currentSong
      Songs = append(Songs, currentSong)
      previousSong = currentSong
    }
  }
  return nil
}

func FetchCurrentStatus(statusChan chan PlayerStatus) error {
  var currentStatus PlayerStatus
  var previousStatus PlayerStatus
  basePollInterval := 7 * time.Second   // Less frequent than song polling
	slowPollInterval := 30 * time.Second  // When idle
	
	ticker := time.NewTicker(basePollInterval)
	defer ticker.Stop()
  for range ticker.C {
    var pollInterval time.Duration
		if Status.IdleTimer >= 10000 {
			pollInterval = slowPollInterval
		} else {
			pollInterval = basePollInterval
		}
		
		ticker.Reset(pollInterval)

		url := "https://10.0.0.120/httpapi.asp?command=getPlayerStatus"
		resp, err := rateLimitedRequest(url)
    if err != nil {
      fmt.Println("error getting from url:",err)
      continue
    }

    respData, err := io.ReadAll(resp.Body)
    if err != nil {
      fmt.Println("error reading:",err)
      return err
    }

    if err := json.Unmarshal([]byte(respData), &currentStatus); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      return err
    }
    resp.Body.Close()
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
  var url string 
  switch command {
    case "play":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:play"
  case "pause":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:pause"
  case "onepause":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:onepause" // toggle pause/pause
  case "next":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:next"
  case "previous":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:previous" 
  case "stop":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:stop"
    Status.Status ="stop"
  case "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12":
    presetNum, _ := strconv.Atoi(command)
    url = fmt.Sprintf("https://10.0.0.120/httpapi.asp?command=MCUKeyShortClick:%d", presetNum)
  case "shuffle":
    url = "https://10.0.0.120/httpapi.asp?command=setPlayerCmd:loopmode:3"
  }
  resp, err := rateLimitedRequest(url)
  if err != nil {
    return err
  }
  defer resp.Body.Close()
  
  return nil
}








