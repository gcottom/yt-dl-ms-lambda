package convert

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/conf"
)

type ConvertResponse struct {
	TrackData string `json:"trackdata,omitempty"`
	FileName  string `json:"filename,omitempty"`
	Author    string `json:"author,omitempty"`
}
type ConvertedResponse struct {
	TrackConverted bool   `json:"converted,omitempty"`
	TrackData      string `json:"trackdata,omitempty"`
	Error          string `json:"error,omitempty"`
}

func Convert(b []byte, u string) error {
	log.Println("entering convert function")
	/*p := os.Getenv("PATH")
	p = p + ":" + os.Getenv("LAMBDA_TASK_ROOT")
	os.Setenv("PATH", p)
	os.Chmod(path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg"), 0777)*/
	var args = []string{"-i", "pipe:0", "-acodec:a", "libmp3lame", "-b:a", "256k", "-f", "mp3", "-"}
	cmd := exec.Command(path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg"), args...)
	resultBuffer := bytes.NewBuffer(make([]byte, 0)) // pre allocate 5MiB buffer

	cmd.Stderr = os.Stderr    // bind log stream to stderr
	cmd.Stdout = resultBuffer // stdout result will be written here

	stdin, err := cmd.StdinPipe() // Open stdin pipe
	if err != nil {
		log.Println(err)
		return err
	}

	err = cmd.Start() // Start a process on another goroutine
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = stdin.Write(b) // pump audio data to stdin pipe
	if err != nil {
		log.Println(err)
		return err
	}
	err = stdin.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		log.Println(err)
		return err
	}
	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		log.Println(err)
		return err
	}
	out := resultBuffer.Bytes()
	err = Upload(out, u+"-conv")
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("exiting convert function")
	return nil
}

func Upload(out []byte, u string) error {
	log.Println("entering upload function")
	conf := aws.Config{Region: aws.String(conf.Region)}
	sess := session.Must(session.NewSession(&conf))
	uploader := s3manager.NewUploader(sess)

	key := fmt.Sprintf("yt-download-%s", u)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("yt-dl-ui-downloads"),
		Key:    aws.String(key),
		Body:   bytes.NewReader(out),
	})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("exiting upload function")
	return nil
}

func TrackConverted(s3id string) bool {
	conf := aws.Config{Region: aws.String(conf.Region)}
	sess := session.Must(session.NewSession(&conf))
	s3svc := s3.New(sess)
	_, err := s3svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String("yt-dl-ui-downloads"),
		Key:    aws.String(s3id),
	})
	return err == nil
}
