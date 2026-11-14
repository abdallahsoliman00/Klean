package server

import (
	"encoding/base64"
	"net/http"
	"slices"

	"klean/backend/internal/config"
	"klean/backend/internal/prompt"
	"klean/backend/internal/store"
)

// imageBody is an image in a request body: its type and its bytes in base64
// (no data: prefix).
type imageBody struct {
	MediaType string `json:"mediaType"`
	Data      string `json:"data"`
}

const imageTypesHelp = "Only PNG, JPEG, WebP and GIF images can be attached."

// parseImages decodes a message's images and checks their number, size and
// type. The type is read from the bytes: the one the client states only has
// to agree with it.
func parseImages(in []imageBody) ([]store.NewImage, error) {
	if len(in) > config.MaxImagesPerMessage {
		return nil, errorf(http.StatusUnprocessableEntity, "A message can have at most %d images.", config.MaxImagesPerMessage)
	}
	out := make([]store.NewImage, 0, len(in))
	for i, img := range in {
		n := i + 1
		if !slices.Contains(config.ImageTypes, img.MediaType) {
			return nil, errorf(http.StatusUnprocessableEntity, "Image %d has the type %q. %s", n, img.MediaType, imageTypesHelp)
		}
		data, err := base64.StdEncoding.DecodeString(img.Data)
		if err != nil {
			return nil, errorf(http.StatusUnprocessableEntity, "Image %d couldn't be read: its data isn't valid base64.", n)
		}
		if len(data) > config.MaxImageBytes {
			return nil, errorf(http.StatusUnprocessableEntity, "Image %d is too large: an image may be at most %d MB.", n, config.ImageMB)
		}
		if http.DetectContentType(data) != img.MediaType {
			return nil, errorf(http.StatusUnprocessableEntity, "Image %d isn't a valid %s file. %s", n, img.MediaType, imageTypesHelp)
		}
		out = append(out, store.NewImage{MediaType: img.MediaType, Data: data})
	}
	return out, nil
}

// turnImages pairs a message's images with their bytes, as the model takes them.
func turnImages(images []store.Image, data map[string][]byte) []prompt.Image {
	var out []prompt.Image
	for _, img := range images {
		out = append(out, prompt.Image{MediaType: img.MediaType, Data: base64.StdEncoding.EncodeToString(data[img.ID])})
	}
	return out
}

// getImage serves an image's bytes. nosniff keeps the browser to the stored
// type, which was checked against the bytes when the image was saved.
func (s *Server) getImage(w http.ResponseWriter, r *http.Request) {
	img, err := s.db.GetImage(r.PathValue("id"))
	if err != nil || img == nil {
		reply(w, 0, nil, orNotFound(err, "Image not found."))
		return
	}
	h := w.Header()
	h.Set("Content-Type", img.MediaType)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", config.ImageCacheControl)
	_, _ = w.Write(img.Data)
}
