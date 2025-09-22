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
var headphoneMode bool
var url string;


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
      if Status.Status == "stop" {
        continue
      }
      if Status.IdleTimer >= 10000 {
        pollInterval = slowPollInterval
      } else {
        pollInterval = basePollInterval
      }
      // Reset ticker if interval changed
      ticker.Reset(pollInterval)
      
      if headphoneMode {
        url = "https://10.0.0.119/httpapi.asp?command=getMetaInfo"
      } else {
        url = "https://10.0.0.60/httpapi.asp?command=getMetaInfo"
      }
      resp, err := rateLimitedRequest(url)
      if err != nil {
        fmt.Println("Error getting from URL %v\n", err)
        continue
      }

      if resp == nil {
        fmt.Println("Status response is nil")
        continue
      }
      if resp.Body == nil {
        fmt.Println("Status response body is nil")
        continue
      }

      respData, err := io.ReadAll(resp.Body)
      resp.Body.Close()
      if err != nil {
        fmt.Println("Error reading json: ", err)
        continue
      }
      currentSong = Response{}
      currentSong.MetaData.Album = "unknown"
      currentSong.MetaData.Title = "unknown"
      currentSong.MetaData.Artist = "unknown"
      currentSong.MetaData.AlbumArtURI = "unknown"
      currentSong.MetaData.BitDepth = "unknown"
      currentSong.MetaData.SampleRate = "unknown"

      if err := json.Unmarshal(respData, &currentSong); err != nil {
        fmt.Println("Cannot unmarshal JSON")
        
      }
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
        select {
        case songChan <- currentSong:
          Songs = append(Songs, currentSong)
          previousSong = currentSong
        }
      }
    }
  return nil
}

func FetchCurrentStatus(statusChan chan PlayerStatus) error {
  headphoneMode = false
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

		if headphoneMode {
        url = "https://10.0.0.119/httpapi.asp?command=getPlayerStatus"
      } else {
        url = "https://10.0.0.60/httpapi.asp?command=getPlayerStatus"
      }
    resp, err := rateLimitedRequest(url)
    if err != nil {
      fmt.Println("error getting from url:",err)
      continue
    }
    if resp == nil {
			fmt.Println("Status response is nil")
			continue
		}
    if resp.Body == nil {
			fmt.Println("Status response body is nil")
			continue
		}

    respData, err := io.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
      fmt.Println("error reading:",err)
      return err
    }

    if err := json.Unmarshal([]byte(respData), &currentStatus); err != nil {
      fmt.Println("Cannot unmarshal JSON")
      return err
    }
    if currentStatus != previousStatus {
      Status = currentStatus
      select {
      case statusChan <- currentStatus:
        previousStatus = currentStatus
      }
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
  cmdUrl := "https://10.0.0.60/httpapi.asp?command="
  if headphoneMode {
    cmdUrl = "https://10.0.0.119/httpapi.asp?command="
  }
  switch command {
  case "play":
    cmdUrl += "setPlayerCmd:play"
  case "pause":
    cmdUrl += "setPlayerCmd:pause"
  case "onepause":
    cmdUrl += "setPlayerCmd:onepause" // toggle pause/pause
  case "next":
    cmdUrl += "setPlayerCmd:next"
  case "previous":
    cmdUrl += "setPlayerCmd:previous" 
  case "stop":
    cmdUrl += "setPlayerCmd:stop"
  case "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12":
    presetNum, _ := strconv.Atoi(command)
    cmdUrl += fmt.Sprintf("MCUKeyShortClick:%d", presetNum)
  case "shuffle":
    cmdUrl += "setPlayerCmd:loopmode:3"
  case "headphoneMode":
    headphoneMode = true
  case "speakersMode":
    headphoneMode = false
  }
  resp, err := rateLimitedRequest(cmdUrl)
  if err != nil {
    return err
  }
  defer resp.Body.Close()
  
  return nil
}








