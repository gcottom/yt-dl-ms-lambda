package meta

import (
	"context"
	"os"
	"strings"

	spotify "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

type TrackMeta struct {
	Title    string `json:"title,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
	AlbumArt string `json:"albumart,omitempty"`
	Genre    string `json:"genre,omitempty"`
	Year     string `json:"year,omitempty"`
	Bpm      string `json:"bpm,omitempty"`
}

type GetMetaResponse struct {
	Results []TrackMeta `json:"results,omitempty"`
}

func GetMetaFromSongAndArtist(songName string, artist string) ([]TrackMeta, error) {
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		return nil, err
	}
	searchTerm := songName + "artist:" + artist
	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)
	results, err := client.Search(ctx, searchTerm, spotify.SearchTypeTrack)
	if err != nil {
		return nil, err
	}
	resultMeta := processMeta(results)
	return resultMeta, nil
}
func processMeta(results *spotify.SearchResult) []TrackMeta {
	resultMeta := []TrackMeta{}
	resultMap := map[spotify.ID]TrackMeta{}
	for _, track := range results.Tracks.Tracks {
		var albumImage string
		if len(track.Album.Images) > 0 {
			albumImage = track.Album.Images[0].URL
		}
		album := track.Album.Name
		var artist string
		for _, art := range track.Artists {
			artist += art.Name + ", "
		}
		artist = artist[:(strings.LastIndex(artist, ", "))] + strings.Replace(artist[(strings.LastIndex(artist, ", ")):], ", ", "", 1)
		song := track.Name
		id := track.ID
		outMeta := TrackMeta{albumImage, album, artist, song, "", "", ""}
		resultMap[id] = outMeta
	}
	for _, v := range resultMap {
		resultMeta = append(resultMeta, v)
	}

	return resultMeta
}
