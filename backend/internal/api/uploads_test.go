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
	server := api.NewServer(logger, fake, false, uploadsDir).Routes()

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

		// The product now points at the file, and the file exists on disk.
		if fake.products[0].ImageURL == "" {
			t.Error("product image_url not updated")
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
}
