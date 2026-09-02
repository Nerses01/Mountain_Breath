package api_test

import (
	"bytes"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Tiny but genuine PNG file (8-byte signature + minimal chunks) — enough
// for http.DetectContentType to say image/png.
var pngBytes = []byte{
	0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A,
	0, 0, 0, 13, 'I', 'H', 'D', 'R',
	0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0,
	0x1F, 0x15, 0xC4, 0x89,
	0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xAE, 0x42, 0x60, 0x82,
}

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}

func TestUploadProductImage(t *testing.T) {
	uploadsDir := t.TempDir()
	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 1, Slug: "tea", Name: "Tea"}}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := api.NewServer(logger, fake, false, uploadsDir, api.Options{}).Routes()

	doUpload := func(path, filename string, content []byte) *httptest.ResponseRecorder {
		body, contentType := multipartBody(t, "image", filename, content)
		req := httptest.NewRequest(http.MethodPost, path, body)
		req.Header.Set("Content-Type", contentType)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		return rec
	}

	t.Run("accepts a real png and stores it", func(t *testing.T) {
		rec := doUpload("/api/v1/admin/products/1/image", "photo.png", pngBytes)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}

		// The gallery gained the row, and the file exists on disk.
		if len(fake.savedImages) != 1 {
			t.Errorf("gallery rows = %d, want 1", len(fake.savedImages))
		}
		files, err := os.ReadDir(uploadsDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 stored file, found %d", len(files))
		}
		if filepath.Ext(files[0].Name()) != ".png" {
			t.Errorf("stored extension = %q, want .png (sniffed, not from filename)", files[0].Name())
		}
	})

	t.Run("rejects a text file pretending to be an image", func(t *testing.T) {
		rec := doUpload("/api/v1/admin/products/1/image", "totally-real.png", []byte("#!/bin/sh\nrm -rf /\n"))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 — content sniffing must ignore the filename", rec.Code)
		}
	})

	t.Run("unknown product gets 404 and no orphan file", func(t *testing.T) {
		before, _ := os.ReadDir(uploadsDir)
		rec := doUpload("/api/v1/admin/products/999/image", "photo.png", pngBytes)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
		after, _ := os.ReadDir(uploadsDir)
		if len(after) != len(before) {
			t.Errorf("orphan file left behind: %d -> %d files", len(before), len(after))
		}
	})

	t.Run("a full gallery answers 409 and no orphan file", func(t *testing.T) {
		// One photo landed in the first subtest; fill the remaining slots.
		for len(fake.savedImages) < domain.MaxGalleryImages {
			if rec := doUpload("/api/v1/admin/products/1/image", "photo.png", pngBytes); rec.Code != http.StatusOK {
				t.Fatalf("filling the gallery: status = %d", rec.Code)
			}
		}
		before, _ := os.ReadDir(uploadsDir)
		rec := doUpload("/api/v1/admin/products/1/image", "photo.png", pngBytes)
		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409 (body: %s)", rec.Code, rec.Body.String())
		}
		after, _ := os.ReadDir(uploadsDir)
		if len(after) != len(before) {
			t.Errorf("orphan file left behind: %d -> %d files", len(before), len(after))
		}
	})

	t.Run("an oversize upload answers 413", func(t *testing.T) {
		// One byte past the cap. The size gate runs BEFORE sniffing, so the
		// content being garbage is irrelevant — that is the point: nobody
		// buffers 5 MB just to learn it was never allowed.
		rec := doUpload("/api/v1/admin/products/1/image", "big.png", make([]byte, maxUploadTestBytes))
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status = %d, want 413", rec.Code)
		}
	})
}

// maxUploadTestBytes clears the 5 MB image cap including multipart framing.
const maxUploadTestBytes = 5<<20 + 1

// Minimal magic numbers, enough for the server's sniffing to classify them.
var (
	// EBML header — what both .webm and .mkv start with; Go's sniff table
	// calls it video/webm.
	webmBytes = []byte{0x1A, 0x45, 0xDF, 0xA3, 1, 2, 3, 4, 5, 6, 7, 8}
	// An MP4 whose ftyp brand is "mp42" — one of the two brands Go's sniff
	// table recognizes as video/mp4.
	mp42Bytes = []byte("\x00\x00\x00\x14ftypmp42\x00\x00\x00\x00mp42")
	// An MP4 branded "isom" — the common real-world case Go's table does NOT
	// recognize; this exercises the handler's ftyp fallback.
	isomBytes = []byte("\x00\x00\x00\x14ftypisom\x00\x00\x00\x00isom")
)

func TestUploadProductVideo(t *testing.T) {
	uploadsDir := t.TempDir()
	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 1, Slug: "tea", Name: "Tea"}}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := api.NewServer(logger, fake, false, uploadsDir, api.Options{}).Routes()

	doUpload := func(path, filename string, content []byte) *httptest.ResponseRecorder {
		body, contentType := multipartBody(t, "video", filename, content)
		req := httptest.NewRequest(http.MethodPost, path, body)
		req.Header.Set("Content-Type", contentType)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		return rec
	}

	countFiles := func() int {
		files, err := os.ReadDir(uploadsDir)
		if err != nil {
			t.Fatal(err)
		}
		return len(files)
	}

	t.Run("accepts a webm and stores it with the sniffed extension", func(t *testing.T) {
		rec := doUpload("/api/v1/admin/products/1/video", "clip.mov", webmBytes)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
		files, _ := os.ReadDir(uploadsDir)
		if len(files) != 1 || filepath.Ext(files[0].Name()) != ".webm" {
			t.Errorf("stored file = %v, want one .webm (sniffed, not the .mov the client claimed)", files)
		}
		if fake.savedVideo == nil {
			t.Fatal("store never saw the video")
		}
	})

	t.Run("a second video answers 409 and no orphan file", func(t *testing.T) {
		before := countFiles()
		rec := doUpload("/api/v1/admin/products/1/video", "clip.webm", webmBytes)
		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
		if countFiles() != before {
			t.Error("orphan file left behind by the refused second video")
		}
	})

	t.Run("accepts an mp42-branded mp4", func(t *testing.T) {
		fake.savedVideo = nil // free the slot
		rec := doUpload("/api/v1/admin/products/1/video", "clip.mp4", mp42Bytes)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("accepts an isom-branded mp4 via the ftyp fallback", func(t *testing.T) {
		// Go's own sniff table answers application/octet-stream for this
		// file; without the handler's fallback every phone-recorded MP4
		// would be rejected.
		fake.savedVideo = nil
		before, _ := os.ReadDir(uploadsDir)
		seen := make(map[string]bool, len(before))
		for _, f := range before {
			seen[f.Name()] = true
		}

		rec := doUpload("/api/v1/admin/products/1/video", "clip.mp4", isomBytes)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (body: %s)", rec.Code, rec.Body.String())
		}

		// ReadDir sorts by (random) name, so "the last entry" is not "the
		// newest" — diff the listing to find what THIS upload stored.
		after, _ := os.ReadDir(uploadsDir)
		for _, f := range after {
			if !seen[f.Name()] && filepath.Ext(f.Name()) != ".mp4" {
				t.Errorf("stored %q, want an .mp4 extension", f.Name())
			}
		}
	})

	t.Run("rejects an image posted to the video route", func(t *testing.T) {
		fake.savedVideo = nil
		rec := doUpload("/api/v1/admin/products/1/video", "clip.mp4", pngBytes)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 — the whitelists must not bleed into each other", rec.Code)
		}
	})

	t.Run("unknown product gets 404 and no orphan file", func(t *testing.T) {
		before := countFiles()
		rec := doUpload("/api/v1/admin/products/999/video", "clip.webm", webmBytes)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
		if countFiles() != before {
			t.Error("orphan file left behind")
		}
	})
}
