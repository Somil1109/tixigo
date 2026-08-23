package httpapi

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/tixigo/tixigo-api/internal/media"
	"github.com/tixigo/tixigo-api/internal/movie"
	"net/http"
)

type organiserMovieHandler struct {
	movies *movie.Store
	media  *media.Cloudinary
}

func (h organiserMovieHandler) submit(w http.ResponseWriter, r *http.Request) {
	if err := h.movies.Submit(r.Context(), chi.URLParam(r, "movieID"), accessClaims(r).Subject); err != nil {
		writeJSON(w, 409, map[string]string{"message": "Movie cannot be submitted."})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Movie submitted for approval."})
}

func (h organiserMovieHandler) create(w http.ResponseWriter, r *http.Request) {
	var draft movie.Draft
	if json.NewDecoder(r.Body).Decode(&draft) != nil {
		writeJSON(w, 400, map[string]string{"message": "Invalid movie payload."})
		return
	}
	if err := draft.Validate(); err != nil {
		writeJSON(w, 400, map[string]string{"message": err.Error()})
		return
	}
	created, err := h.movies.CreateDraft(r.Context(), draft, accessClaims(r).Subject)
	if err != nil {
		writeJSON(w, 500, map[string]string{"message": "Could not create movie draft."})
		return
	}
	writeJSON(w, 201, map[string]any{"data": created})
}
func (h organiserMovieHandler) uploadPoster(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	file, header, err := r.FormFile("poster")
	if err != nil {
		writeJSON(w, 400, map[string]string{"message": "Poster image is required."})
		return
	}
	defer file.Close()
	url, err := h.media.UploadPoster(r.Context(), header.Filename, file)
	if err != nil {
		writeJSON(w, 502, map[string]string{"message": "Poster upload failed."})
		return
	}
	writeJSON(w, 201, map[string]any{"data": map[string]string{"url": url}})
}
