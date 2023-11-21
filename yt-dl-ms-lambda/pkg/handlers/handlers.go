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
	"github.com/google/uuid"
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
	b, title, author, err := yt.Download(yturl)
	if err != nil {
		log.Println(err.Error())
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	u := uuid.New()
	err = convert.Upload(b, u.String())
	if err != nil {
		log.Println(err)
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	queueUrl := "https://sqs.us-east-2.amazonaws.com/112343695294/yt-dl-ms-conversion.fifo"
	conf := aws.Config{Region: aws.String(conf.Region)}
	sess := session.Must(session.NewSession(&conf))
	sqsClient := sqs.New(sess)
	if err != nil {
		log.Println(err)
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	_, err = sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:       &queueUrl,
		MessageBody:    aws.String(u.String()),
		MessageGroupId: aws.String("convertGroup"),
	})
	if err != nil {
		log.Println(err)
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	return apiResponse(http.StatusOK, convert.ConvertResponse{TrackData: "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-" + u.String() + "-conv", FileName: title, Author: author})
}

func GetTrackConvertedHandler(req Request) (*Response, error) {
	s3id := req.PathParameters["s3id"]
	if !convert.TrackConverted(s3id) {
		return apiResponse(http.StatusOK, convert.ConvertedResponse{TrackConverted: false, Error: "File Conversion Not Completed Yet!"})
	}
	return apiResponse(http.StatusOK, convert.ConvertedResponse{TrackConverted: true, TrackData: "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-" + s3id})
}

func SetMetaHandler(req Request) (*Response, error) {
	log.Println("Entering setMetaHandler")
	var reqdata meta.SetTrackMetaRequest
	err := json.Unmarshal([]byte(req.Body), &reqdata)
	if err != nil {
		log.Println("Error unmarshalling json")
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	objectKey := strings.ReplaceAll(reqdata.TrackUrl, "https://yt-dl-ui-downloads.s3.us-east-2.amazonaws.com/yt-download-", "")
	key := fmt.Sprintf("yt-download-%s", objectKey)

	temppath := "/tmp/" + objectKey + ".mp3"
	log.Println("temppath: " + temppath)
	file, err := os.Create(temppath)
	if err != nil {
		log.Println("Error creating temp file")
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
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
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	trackWithMeta, sanTrackName, err := meta.SaveMeta(temppath, reqdata.Title, reqdata.Artist, reqdata.Album, reqdata.AlbumArt)
	if err != nil {
		log.Println("Error saving meta")
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("yt-dl-ui-downloads"),
		Key:    aws.String(key),
		Body:   bytes.NewReader(trackWithMeta),
	})
	if err != nil {
		log.Println("Error uploading to S3")
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}

	return apiResponse(http.StatusOK, meta.SetTrackMetaResponse{FileName: sanTrackName})
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
func GetMetaInitHandler(req Request) (*Response, error) {
	qp := req.PathParameters["data"]
	parsedParams, err := url.ParseQuery(qp)
	if err != nil {
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	artistToTitleMap := make(map[string][]string)
	for key, values := range parsedParams {
		artistName := key
		for _, value := range values {
			decodedValue, err := url.QueryUnescape(value)
			if err != nil {
				return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
			}
			artistToTitleMap[artistName] = append(artistToTitleMap[artistName], decodedValue)
		}
	}
	resultMeta := []meta.TrackMeta{}
	for k, v := range artistToTitleMap {
		for _, v1 := range v {
			tMeta, err := meta.GetMetaFromSongAndArtist(v1, k)
			if err != nil {
				return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
			}
			resultMeta = append(resultMeta, tMeta...)
		}
	}
	return apiResponse(http.StatusOK, meta.GetMetaResponse{Results: resultMeta})

}

func UnhandledMethod() (*events.APIGatewayProxyResponse, error) {
	return apiResponse(http.StatusMethodNotAllowed, ErrorMethodNotAllowed)
}
