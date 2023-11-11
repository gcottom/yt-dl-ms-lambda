package meta

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	tag "github.com/gcottom/audiometa"
)

type SetTrackMetaRequest struct {
	TrackUrl string `json:"url,omitempty"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	AlbumArt string `json:"albumart"`
}

type SetTrackMetaResponse struct {
	FileName string `json:"filename,omitempty"`
	Error    string `json:"err,omitempty"`
}

func SaveMeta(trackUrl, title, artist, album, albumart string) ([]byte, string, error) {
	coverFileName := os.TempDir() + fmt.Sprintf("/%s+%s+cover.jpg", artist, title)
	url := albumart
	idTag, err := tag.OpenTag(trackUrl)
	if err != nil {
		fmt.Println("Error opening track in SaveMeta function")
		return nil, "", err
	}
	if url != "" {
		response, err := http.Get(url)
		if err != nil {
			return nil, "", err
		}
		defer response.Body.Close()
		file, err := os.Create(coverFileName)
		if err != nil {
			return nil, "", err
		}
		defer file.Close()
		_, err = io.Copy(file, response.Body)
		if err != nil {
			return nil, "", err
		}
		idTag.SetAlbumArtFromFilePath(coverFileName)
	}

	idTag.SetTitle(title)
	idTag.SetAlbum(album)
	idTag.SetArtist(artist)
	idTag.Save()
	//os.Remove(coverFileName)
	var singleArtist string
	if len(strings.Split(artist, ",")) > 1 {
		singleArtist = strings.Split(artist, ",")[0]
	} else {
		singleArtist = artist
	}
	newFileName := os.TempDir() + "/" + SanitizeFilename(singleArtist) + " - " + SanitizeFilename(title) + ".mp3"
	defer os.Remove(newFileName)
	err = os.Rename(trackUrl, newFileName)
	if err != nil {
		return nil, "", err
	}
	outFile, err := os.ReadFile(newFileName)
	if err != nil {
		return nil, "", err
	}
	return outFile, (SanitizeFilename(singleArtist) + " - " + SanitizeFilename(title) + ".mp3"), nil

}

func SanitizeFilename(fileName string) string {
	// Characters not allowed on mac
	//	:/
	// Characters not allowed on linux
	//	/
	// Characters not allowed on windows
	//	<>:"/\|?*

	// Ref https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file#naming-conventions

	fileName = regexp.MustCompile(`[:/<>\:"\\|?*]`).ReplaceAllString(fileName, "")
	fileName = regexp.MustCompile(`\s+`).ReplaceAllString(fileName, " ")

	return fileName
}
