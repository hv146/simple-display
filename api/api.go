package api

import (
  "fmt"
  "encoding/json"
  "crypto/tls"
  "net/http"
  "log"
  "os"
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


func SendCurrentSong(songChan chan string) error {
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
		os.Exit(1)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var song Response
	if err := json.Unmarshal(respData, &song); err != nil {
		fmt.Println("Cannot unmarshal JSON")
    return err
	}
	//fmt.Println(song)
	song.MetaData.AlbumArtURI = strings.Replace(song.MetaData.AlbumArtURI, "320x320.jpg", "640x640.jpg", 1)
  return nil
}

func FetchCurrentStatus(statusChan chan string) error {
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
		os.Exit(1)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var playerStatus PlayerStatus
	if err := json.Unmarshal([]byte(respData), &playerStatus); err != nil {
		fmt.Println("Cannot unmarshal JSON")
    return err
	}
  return nil
}
