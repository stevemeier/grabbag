package main

// A very minimalistic web-server to handle basic files
// NO SECURITY CHECKS AT ALL !!! USE AT YOUR OWN RISK

import "github.com/gorilla/mux"
import "github.com/DavidGamba/go-getoptions"
import "net/http"
import "log"
import "time"
import "io"
import "os"
import "regexp"
import "strconv"
import "strings"

func main() {
	var directory string
	var listen string
	var methods []string
	opt := getoptions.New()
	opt.StringVar(&directory, "directory", ``, opt.Required())
	opt.StringVar(&listen, "listen", `:8000`)
	opt.StringSliceVar(&methods, "methods", 1, 4)
	remaining, err := opt.Parse(os.Args[1:])

	// Handle empty or unknown options
        if len(os.Args[1:]) == 0 {
                log.Print(opt.Help())
                os.Exit(1)
        }
        if err != nil {
                log.Fatalf("Could not parse options: %s\n", err)
                os.Exit(1)
        }
        if len(remaining) > 0 {
                log.Fatalf("Unsupported parameter: %s\n", remaining)
                os.Exit(1)
        }

	// By default, enable all methods
	if len(methods) == 0 {
		methods = append(methods, "GET")
		methods = append(methods, "POST")
		methods = append(methods, "PUT")
		methods = append(methods, "DELETE")
		methods = append(methods, "HEAD")
	}

	// Change to storage directory
	direrr := os.Chdir(directory)
	if direrr != nil {
		log.Fatal("ERROR: "+direrr.Error())
		os.Exit(1)
	}

	r := mux.NewRouter()
	if contains(methods, "GET") {
		r.HandleFunc("/{key}", GetHandler).Methods("GET")
	}
	if contains(methods, "POST") {
		r.HandleFunc("/{key}", PostHandler).Methods("POST")
	}
	if contains(methods, "DELETE") {
		r.HandleFunc("/{key}", DeleteHandler).Methods("DELETE")
	}
	if contains(methods, "PUT") {
		r.HandleFunc("/{key}", PutHandler).Methods("PUT")
	}
	if contains(methods, "HEAD") {
		r.HandleFunc("/{key}", HeadHandler).Methods("HEAD")
	}

	srv := &http.Server{
        Handler:      r,
        Addr:         listen,
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

func PutHandler (w http.ResponseWriter, r *http.Request) {
	endpoint := mux.Vars(r)["key"]
	if (StartsWithDot(endpoint)) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Check if file exists, error if not (create == POST)
	_, err1 := os.Stat(endpoint)
	if os.IsNotExist(err1) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
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
	log.Println(`PUT`, endpoint, bytes, err2)
	return
}

func contains (slice []string, item string) (bool) {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}

	return false
}

func HeadHandler (w http.ResponseWriter, r *http.Request) {
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

	log.Println(`HEAD`, endpoint)
	return
}
