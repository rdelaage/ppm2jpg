package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/spakin/netpbm"
)

var storageDir string
var baseURL string
var port int

// ConvertPPMToJPEG converts a PPM image to a JPEG image.
func ConvertPPMToJPEG(data []byte) ([]byte, error) {
	img, err := netpbm.Decode(
		bytes.NewReader(data),
		&netpbm.DecodeOptions{
			Target: netpbm.PPM,
			Exact: false,
		},
	)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, nil)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// uploadHandler handles the image upload and conversion.
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Read the uploaded file
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error reading file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	// Convert to JPEG
	jpegData, err := ConvertPPMToJPEG(data)
	if err != nil {
		http.Error(w, "Error converting to JPEG: "+err.Error(), http.StatusInternalServerError)
		return
	}

	hashBytes := sha256.Sum256(jpegData)
	hash := hex.EncodeToString(hashBytes[:])

	err = os.WriteFile(path.Join(storageDir, hash+".jpg"), jpegData, 0750)
	if err != nil {
		http.Error(w, "Error storing the JPEG: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type and write the JPEG image
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, baseURL+"/files/"+hash+".jpg")
}

func main() {
	flag.StringVar(&storageDir, "storage-dir", "storage", "Directory to use for the storage")
	flag.IntVar(&port, "port", 8080, "Port to bind to")
	flag.StringVar(&baseURL, "base-url", "http://localhost:8080", "Base URL")
	flag.Parse()

	err := os.MkdirAll(storageDir, 0750)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create storage directory:", err)
		os.Exit(1)
	}

	http.HandleFunc("/upload", uploadHandler)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(storageDir))))

	fmt.Println(fmt.Sprintf("Starting server on :%d", port))
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting server:", err)
		os.Exit(1)
	}
}
