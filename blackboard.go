package main

// A very minimalistic web-server to handle basic files
// NO SECURITY CHECKS AT ALL !!! USE AT YOUR OWN RISK

import "github.com/gorilla/mux"
import "net/http"
import "log"
import "time"
import "io"
import "os"
import "regexp"
import "strconv"

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/{key}", GetHandler).Methods("GET")
	r.HandleFunc("/{key}", PostHandler).Methods("POST")
	r.HandleFunc("/{key}", DeleteHandler).Methods("DELETE")

	srv := &http.Server{
        Handler:      r,
        Addr:         ":8000",
        // Good practice: enforce timeouts for servers you create!
        WriteTimeout: 5 * time.Second,
        ReadTimeout:  5 * time.Second,
    }

    log.Fatal(srv.ListenAndServe())
}

func GetHandler (w http.ResponseWriter, r *http.Request) {
	endpoint := mux.Vars(r)["key"]
	if (StartsWithDot(endpoint)) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Try to open file
	fh, err := os.Open(endpoint)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer fh.Close()

	// Automagically set Content-Type header
	contentType, err2 := GetFileContentType(fh)
	if err2 == nil {
		w.Header().Set(`Content-Type`, contentType)
	}

	// Get file size for Content-Length header
	stat, err3 := os.Stat(endpoint)
	if err3 == nil {
		w.Header().Set(`Content-Length`, strconv.FormatInt(stat.Size(), 10) )
	}

	// Log & Return
	bytes, err4 := io.Copy(w, fh)
	log.Println(`GET`, endpoint, bytes, err4)

	return
}

func PostHandler (w http.ResponseWriter, r *http.Request) {
	endpoint := mux.Vars(r)["key"]
	if (StartsWithDot(endpoint)) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Check if file exists, create if it does not
	_, err1 := os.Stat(endpoint)
	if os.IsNotExist(err1) {
		os.Create(endpoint)
	}

	// Open for writing
	fh, err := os.OpenFile(endpoint, os.O_WRONLY, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}
	defer fh.Close()

	// Copy data to file
	// Truncate in case previous content was longer than new content
	bytes, err2 := io.Copy(fh, r.Body)
	fh.Truncate(bytes)

	// Log & Return
	log.Println(`POST`, endpoint, bytes, err2)
	return
}

func DeleteHandler (w http.ResponseWriter, r *http.Request) {
	var success bool = false
	endpoint := mux.Vars(r)["key"]
	if (StartsWithDot(endpoint)) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Check if file exists
	_, err1 := os.Stat(endpoint)
	if os.IsNotExist(err1) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Remove file
	err2 := os.Remove(endpoint)
	if err2 == nil {
		success = true
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	log.Println(`DELETE`, endpoint, success)
	return
}

func GetFileContentType(out *os.File) (string, error) {
	// Store current file offset
	offset, _ := out.Seek(0, io.SeekCurrent)

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	// Restore previous offset
	_, _ = out.Seek(offset, 0)

	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func StartsWithDot (path string) (bool) {
	match, _ := regexp.MatchString("^\\.", path)
	return match
}
