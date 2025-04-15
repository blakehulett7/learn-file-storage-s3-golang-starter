package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "only takes jpegs or pngs", err)
		return
	}

	fileExtensions, err := mime.ExtensionsByType(mediaType)
	fileExtension := fileExtensions[0]
	nameData := make([]byte, 32)
	rand.Read(nameData)
	fileName := base64.RawURLEncoding.EncodeToString(nameData)
	filePath := fmt.Sprintf("%v%v", filepath.Join(cfg.assetsRoot, fileName), fileExtension)

	newFile, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err)
	}
	defer newFile.Close()
	_, err = io.Copy(newFile, file)

	// data, err := io.ReadAll(file)

	metadata, err := cfg.db.GetVideo(videoID)
	if metadata.UserID != userID {
		respondWithJSON(w, http.StatusUnauthorized, struct{}{})
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%v/%v", cfg.port, filePath)
	metadata.ThumbnailURL = &thumbnailURL
	cfg.db.UpdateVideo(metadata)

	respondWithJSON(w, http.StatusOK, metadata)
}
