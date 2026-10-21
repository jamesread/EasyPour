package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	auth "github.com/jamesread/httpauthshim"
)

const maxUploadBytes = 10 << 20

// handleUpload accepts POST multipart/form-data with "image" file; requires admin. Saves to imagesDir, returns {"url": "/images/filename"}.
func handleUpload(authCtx *auth.AuthShimContext, imagesDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authorizeUpload(authCtx, r); err != nil {
			writeHTTPError(w, err)
			return
		}
		url, err := saveUploadedImage(r, imagesDir)
		if err != nil {
			writeHTTPError(w, err)
			return
		}
		writeJSON(w, map[string]string{"url": url})
	}
}

func authorizeUpload(authCtx *auth.AuthShimContext, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errMethodNotAllowed
	}
	user := authenticatedUser(authCtx, r)
	if user == nil {
		return errUnauthorized
	}
	if !userInAdminGroup(user) {
		return errForbidden
	}
	return nil
}

func saveUploadedImage(r *http.Request, imagesDir string) (string, error) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return "", errBadRequest
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		return "", &httpError{http.StatusBadRequest, "missing or invalid image field"}
	}
	defer func() { _ = file.Close() }()
	filename := uuid.New().String() + imageExtension(header.Filename)
	if err := writeImageFile(imagesDir, filename, file); err != nil {
		return "", err
	}
	return "/images/" + filename, nil
}

func imageExtension(filename string) string {
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}
	return ".jpg"
}

// writeImageFile writes through os.Root so the client-supplied extension cannot escape imagesDir.
func writeImageFile(imagesDir, filename string, src io.Reader) error {
	if err := os.MkdirAll(imagesDir, 0o750); err != nil {
		return err
	}
	root, err := os.OpenRoot(imagesDir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return createFileInRoot(root, filename, src)
}

func createFileInRoot(root *os.Root, name string, src io.Reader) error {
	dest, err := root.Create(name)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dest, src)
	if err := errors.Join(copyErr, dest.Close()); err != nil {
		_ = root.Remove(name)
		return err
	}
	return nil
}
