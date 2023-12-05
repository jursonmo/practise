package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/skip2/go-qrcode"
)

func generateQRCode(w http.ResponseWriter, r *http.Request) {
	// Generate QR code
	qrCode, err := qrcode.New("Hello, QR Code!", qrcode.Medium)
	if err != nil {
		http.Error(w, "Error generating QR code", http.StatusInternalServerError)
		return
	}

	qrCode.WriteFile(256, "./xx.png")

	// Serve QR code as an image
	w.Header().Set("Content-Type", "image/png")
	png, err := qrCode.PNG(256) //256 * 256
	if err != nil {
		http.Error(w, "Error serving QR code", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(png)
	if err != nil {
		http.Error(w, "Error serving QR code", http.StatusInternalServerError)
		return
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Display a simple HTML page with the QR code
	tmpl, err := template.New("index").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>QR Code Generator</title>
		</head>
		<body>
			<h1>QR Code Generator</h1>
			<img src="/qrcode" alt="QR Code">
		</body>
		</html>
	`)
	if err != nil {
		http.Error(w, "Error rendering HTML", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error rendering HTML", http.StatusInternalServerError)
		return
	}
}

func main() {
	// Define routes
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/qrcode", generateQRCode)

	// Start HTTP server
	port := 8080
	fmt.Printf("Server is running on http://localhost:%d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
