package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

const (
	maxImageBytes = 5 << 20  // 5 MB
	maxVideoBytes = 50 << 20 // 50 MB — a "short clip" cap, not a hosting service
)

// Allowed types, keyed by the SNIFFED content type — the client-supplied
// filename and Content-Type header are attacker-controlled and ignored.
var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var videoExtensions = map[string]string{
	"video/mp4":  ".mp4",
	"video/webm": ".webm",
}

// receiveUpload is the half of an upload every media kind shares: read the
// multipart field, cap the size, sniff the REAL content type against the
// whitelist, and write the bytes under a server-generated name. It answers
// the HTTP error itself and reports ok=false; a caller handles only its own
// store call — and must os.Remove(diskPath) if that call fails, so a
// rejected append does not strand an orphan file.
func (s *Server) receiveUpload(
	w http.ResponseWriter, r *http.Request,
	productID int64, field string, maxBytes int64, allowed map[string]string,
) (url, diskPath string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	file, _, err := r.FormFile(field)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			s.respondError(w, http.StatusRequestEntityTooLarge, "too_large",
				fmt.Sprintf("%s must be under %d MB", field, maxBytes>>20))
			return "", "", false
		}
		s.respondError(w, http.StatusBadRequest, "invalid_upload",
			fmt.Sprintf("multipart field %q is required", field))
		return "", "", false
	}
	defer func() { _ = file.Close() }()

	// Sniff the REAL content type from the first bytes (magic numbers).
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		s.respondError(w, http.StatusBadRequest, "invalid_upload", "could not read uploaded file")
		return "", "", false
	}
	contentType := http.DetectContentType(head[:n])
	// Go's sniff table only says video/mp4 for ftyp brands starting "mp4"
	// (mp41/mp42). Most real MP4s are branded isom/avc1 and sniff as
	// application/octet-stream, so recognize the container shape ourselves:
	// bytes 4..8 of every MP4 spell "ftyp". Harmless for the image handler —
	// its whitelist has no video/mp4 entry to match.
	if contentType == "application/octet-stream" &&
		n >= 12 && bytes.Equal(head[4:8], []byte("ftyp")) {
		contentType = "video/mp4"
	}
	ext, allowedType := allowed[contentType]
	if !allowedType {
		s.respondError(w, http.StatusBadRequest, "unsupported_type",
			fmt.Sprintf("got %s — only %s files are accepted", contentType, extensionList(allowed)))
		return "", "", false
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return "", "", false
	}

	// Server-generated name: no user input ever reaches the filesystem path.
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return "", "", false
	}
	filename := fmt.Sprintf("p%d-%s%s", productID, hex.EncodeToString(suffix), ext)

	diskPath = filepath.Join(s.uploadsDir, filename)
	dst, err := os.Create(diskPath)
	if err != nil {
		s.log.Error("creating upload file", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return "", "", false
	}
	_, copyErr := io.Copy(dst, file)
	// Close BEFORE any possible os.Remove: Windows cannot delete open files.
	if err := dst.Close(); err != nil && copyErr == nil {
		copyErr = err
	}
	if copyErr != nil {
		_ = os.Remove(diskPath)
		s.log.Error("writing upload", "error", copyErr)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return "", "", false
	}

	return "/uploads/" + filename, diskPath, true
}

// extensionList renders a whitelist for an error message ("JPEG, PNG or
// WebP"), derived from the map so the message cannot drift from the check.
func extensionList(allowed map[string]string) string {
	names := map[string]string{
		"image/jpeg": "JPEG", "image/png": "PNG", "image/webp": "WebP",
		"video/mp4": "MP4", "video/webm": "WebM",
	}
	// Deterministic order for tests and admins alike.
	order := []string{"image/jpeg", "image/png", "image/webp", "video/mp4", "video/webm"}
	out := ""
	for _, ct := range order {
		if _, ok := allowed[ct]; !ok {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += names[ct]
	}
	return out
}

func (s *Server) uploadProductID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "product id must be a number")
		return 0, false
	}
	return productID, true
}

// POST /admin/products/{id}/image — multipart upload, field name "image".
func (s *Server) handleUploadProductImage(w http.ResponseWriter, r *http.Request) {
	productID, ok := s.uploadProductID(w, r)
	if !ok {
		return
	}
	url, diskPath, ok := s.receiveUpload(w, r, productID, "image", maxImageBytes, imageExtensions)
	if !ok {
		return
	}

	// An upload APPENDS to the gallery (product_images). The first one a
	// product gets becomes its hero automatically — see store.AddProductImage.
	//
	// Alt text starts empty rather than being invented from the product
	// name: a caption the admin never wrote is worse than a visibly missing
	// one, because it looks finished. The images form is where it gets set.
	img, err := s.store.AddProductImage(r.Context(), productID, url, nil)
	if err != nil {
		_ = os.Remove(diskPath) // don't strand orphan files for refused appends
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
		case errors.Is(err, domain.ErrGalleryFull):
			// 409, not 400: the FILE was fine — it is the gallery's current
			// state the upload conflicts with.
			s.respondError(w, http.StatusConflict, "gallery_full",
				fmt.Sprintf("a product can hold at most %d photos — delete one first", domain.MaxGalleryImages))
		default:
			s.log.Error("saving image", "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]any{
		"image_url":  url,
		"image_id":   img.ID,
		"is_primary": img.IsPrimary,
	})
}

// POST /admin/products/{id}/video — multipart upload, field name "video".
// Fills the product's single video slot; replacing means DELETE (the shared
// images route — the video is a product_images row) and upload again.
func (s *Server) handleUploadProductVideo(w http.ResponseWriter, r *http.Request) {
	productID, ok := s.uploadProductID(w, r)
	if !ok {
		return
	}
	url, diskPath, ok := s.receiveUpload(w, r, productID, "video", maxVideoBytes, videoExtensions)
	if !ok {
		return
	}

	video, err := s.store.AddProductVideo(r.Context(), productID, url)
	if err != nil {
		_ = os.Remove(diskPath)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
		case errors.Is(err, domain.ErrVideoExists):
			s.respondError(w, http.StatusConflict, "video_exists",
				"this product already has a video — delete it first")
		default:
			s.log.Error("saving video", "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]any{
		"video_url": url,
		"video_id":  video.ID,
	})
}
