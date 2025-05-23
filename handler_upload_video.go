package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	maxUploadSize := int64(1 << 30)

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	metadata, err := cfg.db.GetVideo(videoID)
	if metadata.UserID != userID {
		respondWithJSON(w, http.StatusUnauthorized, struct{}{})
		return
	}

	file, header, err := r.FormFile("video")
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "only takes jpegs or pngs", err)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)

	tmpFile.Seek(0, io.SeekStart)

	prefix, err := getVideoAspectRatio(tmpFile.Name())
	nameData := make([]byte, 32)
	rand.Read(nameData)
	fileName := base64.RawURLEncoding.EncodeToString(nameData)

	fileKey := fmt.Sprintf("%v/%v.mp4", prefix, fileName)
	fmt.Println(fileKey)

	processedVideoFilePath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		fmt.Printf("couldn't process video: %v\n", err)
		return
	}

	processedVideo, err := os.Open(processedVideoFilePath)
	if err != nil {
		fmt.Printf("couldn't open processed video: %v\n", err)
		return
	}
	defer os.Remove(processedVideoFilePath)
	defer processedVideo.Close()

	cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        processedVideo,
		ContentType: &mediaType,
	})

	videoURL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, fileKey)
	metadata.VideoURL = &videoURL
	cfg.db.UpdateVideo(metadata)
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	buffer := &bytes.Buffer{}
	cmd.Stdout = buffer
	cmd.Run()
	output := struct {
		Streams []struct {
			Height int `json:"height"`
			Width  int `json:"width"`
		}
	}{}
	json.Unmarshal(buffer.Bytes(), &output)
	video := output.Streams[0]
	fmt.Println(video)

	if video.Width == 16*video.Height/9 {
		return "landscape", nil
	}

	if video.Height == 16*video.Width/9 {
		return "portrait", nil
	}

	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputPath, nil
}
