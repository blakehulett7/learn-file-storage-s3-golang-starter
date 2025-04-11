package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

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
	mediaType := header.Header.Get("Content-Type")
	data, err := io.ReadAll(file)

	encoded := base64.StdEncoding.EncodeToString(data)
	imageCode := fmt.Sprintf("data:%v;base64,%v", mediaType, encoded)

	metadata, err := cfg.db.GetVideo(videoID)
	if metadata.UserID != userID {
		respondWithJSON(w, http.StatusUnauthorized, struct{}{})
		return
	}

	thumbnail := thumbnail{
		data:      data,
		mediaType: mediaType,
	}

	videoThumbnails[videoID] = thumbnail

	metadata.ThumbnailURL = &imageCode
	cfg.db.UpdateVideo(metadata)

	respondWithJSON(w, http.StatusOK, metadata)
}
