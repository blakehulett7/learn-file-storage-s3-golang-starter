package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

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

	fmt.Println(r.Form)
	file, header, err := r.FormFile("video")
	fmt.Println(file)
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

	nameData := make([]byte, 32)
	rand.Read(nameData)
	fileName := base64.RawURLEncoding.EncodeToString(nameData)
	fileKey := fmt.Sprintf("%v.mp4", fileName)

	cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        tmpFile,
		ContentType: &mediaType,
	})

	videoURL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, fileKey)
	metadata.VideoURL = &videoURL
	cfg.db.UpdateVideo(metadata)
}
