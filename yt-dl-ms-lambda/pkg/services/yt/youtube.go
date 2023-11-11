package yt

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/kkdai/youtube/v2"
)

func Download(id string) ([]byte, string, error) {
	id = strings.Replace(id, "&feature=share", "", 1)
	id = strings.Replace(id, "https://music.youtube.com/watch?v=", "", 1)
	id = strings.Replace(id, "https://www.music.youtube.com/watch?v=", "", 1)
	id = strings.Replace(id, "https://www.youtube.com/watch?v", "", 1)
	id = strings.Replace(id, "https://youtube.com/watch?v", "", 1)
	id = strings.Split(id, "&")[0]
	videoID := id // Replace with the YouTube video ID you want to download

	// Create a new YouTube client
	client := youtube.Client{}

	// Get the video info
	videoInfo, err := client.GetVideo(videoID)
	if err != nil {
		err = fmt.Errorf("failed to get video info: %v", err)
		return nil, "", err
	}
	// Find the best audio format available
	bestFormat := getBestAudioFormat(videoInfo.Formats.Type("audio"))
	if bestFormat == nil {
		err = fmt.Errorf("no audio formats found for the video")
		return nil, "", err
	}
	stream, _, err := client.GetStream(videoInfo, bestFormat)
	if err != nil {
		err = fmt.Errorf("no Stream found")
		return nil, "", err
	}
	title := SanitizeFilename(videoInfo.Title)
	b, err := io.ReadAll(stream)
	if err != nil {
		err = fmt.Errorf("unable to copy stream data to file object: %v", err)
		return nil, "", err
	}
	return b, title, nil
}

// getBestAudioFormat finds the best audio format from a list of formats
func getBestAudioFormat(formats youtube.FormatList) *youtube.Format {
	var bestFormat *youtube.Format
	maxBitrate := 0

	for _, format := range formats {
		if format.Bitrate > maxBitrate {
			bestFormat = &format
			maxBitrate = format.Bitrate
		}
	}
	fmt.Println(bestFormat.QualityLabel)
	return bestFormat
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
