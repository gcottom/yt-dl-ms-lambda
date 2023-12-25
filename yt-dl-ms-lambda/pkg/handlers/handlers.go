package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/conf"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/services/convert"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/services/meta"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/services/yt"
)

var ErrorMethodNotAllowed = "method Not allowed"

type Request = events.APIGatewayProxyRequest
type Response = events.APIGatewayProxyResponse

type ErrorBody struct {
	ErrorMsg *string `json:"error,omitempty"`
}

func GetTrackHandler(req Request) (*Response, error) {
	yturl := req.PathParameters["videoid"]
	log.Println(yturl)
	//if track already exists, don't download it from yt again, just use the cached file
	if convert.TrackConverted("yt-download-" + yturl + "-conv") {
		title, author, err := yt.GetInfo(yturl)
		if err != nil {
			log.Println(err)
			return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		return ApiResponse(http.StatusOK, convert.ConvertResponse{TrackData: "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-" + yturl + "-conv", FileName: title, Author: author})
	}
	b, title, author, err := yt.Download(yturl)
	if err != nil {
		log.Println(err.Error())
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	err = convert.Upload(b, yturl)
	if err != nil {
		log.Println(err)
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	queueUrl := "https://sqs.us-east-2.amazonaws.com/112343695294/yt-dl-ms-conversion.fifo"
	conf := aws.Config{Region: aws.String(conf.Region)}
	sess := session.Must(session.NewSession(&conf))
	sqsClient := sqs.New(sess)
	if err != nil {
		log.Println(err)
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	_, err = sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:       &queueUrl,
		MessageBody:    aws.String(yturl),
		MessageGroupId: aws.String("convertGroup"),
	})
	if err != nil {
		log.Println(err)
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	return ApiResponse(http.StatusOK, convert.ConvertResponse{TrackData: "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-" + yturl + "-conv", FileName: title, Author: author})
}

func GetTrackConvertedHandler(req Request) (*Response, error) {
	s3id := req.PathParameters["s3id"]
	if !convert.TrackConverted(s3id) {
		return ApiResponse(http.StatusOK, convert.ConvertedResponse{TrackConverted: false, Error: "File Conversion Not Completed Yet!"})
	}
	return ApiResponse(http.StatusOK, convert.ConvertedResponse{TrackConverted: true, TrackData: "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-" + s3id})
}

func SetMetaHandler(req Request) (*Response, error) {
	log.Println("Entering setMetaHandler")
	var reqdata meta.SetTrackMetaRequest
	err := json.Unmarshal([]byte(req.Body), &reqdata)
	if err != nil {
		log.Println("Error unmarshalling json")
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	objectKey := strings.ReplaceAll(reqdata.TrackUrl, "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-", "")
	key := fmt.Sprintf("yt-download-%s", objectKey)

	temppath := "/tmp/" + objectKey + ".mp3"
	log.Println("temppath: " + temppath)
	file, err := os.Create(temppath)
	if err != nil {
		log.Println("Error creating temp file")
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	defer file.Close()

	conf := aws.Config{Region: aws.String(conf.Region)}
	sess := session.Must(session.NewSession(&conf))

	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String("yt-dl-ui-downloads"),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Println("Error downloading file")
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	trackWithMeta, sanTrackName, err := meta.SaveMeta(temppath, reqdata.Title, reqdata.Artist, reqdata.Album, reqdata.AlbumArt)
	if err != nil {
		log.Println("Error saving meta")
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("yt-dl-ui-downloads"),
		Key:    aws.String(key),
		Body:   bytes.NewReader(trackWithMeta),
	})
	if err != nil {
		log.Println("Error uploading to S3")
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	return ApiResponse(http.StatusOK, meta.SetTrackMetaResponse{FileName: sanTrackName})
}
func ConvertTrackHandler(sqsEvent events.SQSEvent) error {
	for _, r := range sqsEvent.Records {
		msg := r.Body
		key := fmt.Sprintf("yt-download-%s", msg)
		conf := aws.Config{Region: aws.String(conf.Region)}
		sess := session.Must(session.NewSession(&conf))
		buf := aws.NewWriteAtBuffer([]byte{})

		downloader := s3manager.NewDownloader(sess)
		_, err := downloader.Download(buf, &s3.GetObjectInput{
			Bucket: aws.String("yt-dl-ui-downloads"),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Println("Error downloading file from s3")
			return err
		}
		err = convert.Convert(buf.Bytes(), msg)
		if err != nil {
			log.Println("Error downloading file from s3")
			return err
		}
	}
	return nil
}
func GetMetaHandler(req Request) (*Response, error) {
	resultMeta := []meta.TrackMeta{}
	if req.QueryStringParameters["ams"] == "true" {
		author, err := url.QueryUnescape(req.QueryStringParameters["author"])
		if err != nil {
			return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		title, err := url.QueryUnescape(req.QueryStringParameters["title"])
		if err != nil {
			return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		m := meta.GetArtistTitleCombos(title, author)
		for art, v := range m {
			for _, tit := range v {
				tMeta, err := meta.GetMetaFromSongAndArtist(tit, art)
				if err != nil {
					return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
				}
				absolute_match_found, absolute_match := meta.Find_absolute_match(tMeta, art, tit)
				if absolute_match_found {
					return ApiResponse(http.StatusOK, meta.GetMetaResponse{AbsoluteMatchFound: true, AbsoluteMatchMeta: absolute_match})
				}
				resultMeta = append(resultMeta, tMeta...)
			}
		}
		resultMeta = meta.Filter_results(resultMeta)
		return ApiResponse(http.StatusOK, meta.GetMetaResponse{AbsoluteMatchFound: false, Results: resultMeta})
	}

	artist, err := url.QueryUnescape(req.QueryStringParameters["artist"])
	if err != nil {
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	title, err := url.QueryUnescape(req.QueryStringParameters["title"])
	if err != nil {
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	tMeta, err := meta.GetMetaFromSongAndArtist(title, artist)
	if err != nil {
		return ApiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	resultMeta = append(resultMeta, tMeta...)

	return ApiResponse(http.StatusOK, meta.GetMetaResponse{Results: resultMeta})

}

func UnhandledMethod() (*events.APIGatewayProxyResponse, error) {
	return ApiResponse(http.StatusMethodNotAllowed, ErrorMethodNotAllowed)
}
