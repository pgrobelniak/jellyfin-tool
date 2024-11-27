package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/namsral/flag"
)

type JellyfinItem struct {
	Name string
	Id   string
}

type JellyfinItems struct {
	Items []JellyfinItem
}

type JellyfinMediaStream struct {
	Language string
	Type     string
}

type JellyfinPlaybackSource struct {
	MediaStreams []JellyfinMediaStream
}

type JellyfinPlaybackInfo struct {
	MediaSources []JellyfinPlaybackSource
}

var jellyfinAddress, jellyfinToken, libraryId string

func makeRequest(method string, path string, in interface{}, out interface{}) (string, error) {
	url := "https://" + jellyfinAddress + ":8920/" + path
	buffer := new(bytes.Buffer)
	if in != nil {
		if err := json.NewEncoder(buffer).Encode(in); err != nil {
			return "", err
		}
	}
	req, err := http.NewRequest(method, url, buffer)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Token", jellyfinToken)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if out != nil {
		err = json.Unmarshal(bytes, out)
	}
	return string(bytes), err
}

func hasPolishAudio(id string) (bool, error) {
	var res JellyfinPlaybackInfo
	_, err := makeRequest("GET", "Items/"+id+"/PlaybackInfo", nil, &res)
	if err != nil {
		return false, err
	}
	for _, source := range res.MediaSources {
		for _, stream := range source.MediaStreams {
			if stream.Type == "Audio" && stream.Language == "pol" {
				return true, nil
			}
		}
	}
	return false, nil
}

func main() {
	flag.StringVar(&jellyfinAddress, "jellyfin-address", "", "")
	flag.StringVar(&jellyfinToken, "jellyfin-token", "", "")
	flag.StringVar(&libraryId, "library-id", "", "")
	flag.Parse()
	var col JellyfinItem
	_, err := makeRequest("POST", "Collections?name=Lektor", nil, &col)
	if err != nil {
		log.Fatal(err)
	}
	var res JellyfinItems
	_, err = makeRequest("GET", "Items?ParentId="+libraryId, nil, &res)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range res.Items {
		pol, _ := hasPolishAudio(item.Id)
		if pol {
			log.Print(item.Name)
			_, err = makeRequest("POST", "Collections/"+col.Id+"/Items?ids="+item.Id, nil, nil)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
