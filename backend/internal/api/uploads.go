package api

import (
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

const maxImageBytes = 5 << 20 // 5 MB

// Allowed types, keyed by the SNIFFED content type — the client-supplied
// filename and Content-Type header are attacker-controlled and ignored.
var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// POST /admin/products/{id}/image — multipart upload, field name "image".
func (s *Server) handleUploadProductImage(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "product id must be a number")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes)
	file, _, err := r.FormFile("image")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			s.respondError(w, http.StatusRequestEntityTooLarge, "too_large", "image must be under 5 MB")
			return
		}
		s.respondError(w, http.StatusBadRequest, "invalid_upload", "multipart field \"image\" is required")
		return
	}
	defer func() { _ = file.Close() }()

	// Sniff the REAL content type from the first bytes (magic numbers).
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		s.respondError(w, http.StatusBadRequest, "invalid_upload", "could not read uploaded file")
		return
	}
	contentType := http.DetectContentType(head[:n])
	ext, allowed := imageExtensions[contentType]
	if !allowed {
		s.respondError(w, http.StatusBadRequest, "unsupported_type",
			fmt.Sprintf("got %s — only JPEG, PNG or WebP images are accepted", contentType))
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// Server-generated name: no user input ever reaches the filesystem path.
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	filename := fmt.Sprintf("p%d-%s%s", productID, hex.EncodeToString(suffix), ext)

	path := filepath.Join(s.uploadsDir, filename)
	dst, err := os.Create(path)
	if err != nil {
		s.log.Error("creating upload file", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	_, copyErr := io.Copy(dst, file)
	// Close BEFORE any possible os.Remove: Windows cannot delete open files.
	if err := dst.Close(); err != nil && copyErr == nil {
		copyErr = err
	}
	if copyErr != nil {
		_ = os.Remove(path)
		s.log.Error("writing upload", "error", copyErr)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	url := "/uploads/" + filename

	// E3: an upload now APPENDS to the gallery (product_images) instead of
	// overwriting one image_url. The first one a product gets becomes its
	// hero automatically — see store.AddProductImage.
	//
	// Alt text starts empty rather than being invented from the product
	// name: a caption the admin never wrote is worse than a visibly missing
	// one, because it looks finished. The images form is where it gets set.
	img, err := s.store.AddProductImage(r.Context(), productID, url, nil)
	if err != nil {
		_ = os.Remove(path) // don't strand orphan files for missing products
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
			return
		}
		s.log.Error("saving image", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// products.image_url is still written, and still the column the shop
	// grid reads. It is dropped in migration 000015 once every read path
	// uses the gallery — the same add-backfill-then-drop sequence 000007
	// used, and the reason this one write does double duty for now.
	if img.IsPrimary {
		if err := s.store.UpdateProductImage(r.Context(), productID, url); err != nil {
			s.log.Error("syncing legacy image url", "error", err)
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]any{
		"image_url":  url,
		"image_id":   img.ID,
		"is_primary": img.IsPrimary,
	})
}
