package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"

    "github.com/jung-kurt/gofpdf/v2"
    "github.com/sirupsen/logrus"
)

// PaymentData represents the data sent in the payment form
type PaymentData struct {
    UserID         string `json:"user_id"`
    CardNumber     string `json:"cardNumber"`
    ExpirationDate string `json:"expirationDate"`
    CVV            string `json:"cvv"`
    Name           string `json:"name"`
    Address        string `json:"address"`
}

var logger = logrus.New()

func paymentFormHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("payment_form.html")
    if err != nil {
        logger.WithFields(logrus.Fields{
            "action": "paymentFormHandler",
            "status": "error",
            "error":  err.Error(),
        }).Error("Error parsing template file 'payment_form.html'")
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    logger.WithFields(logrus.Fields{
        "action": "paymentFormHandler",
        "status": "success",
    }).Info("User on the page.")
    tmpl.Execute(w, nil)
}

func generatePDFReceipt(paymentData PaymentData) (string, error) {
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.AddPage()
    pdf.SetFont("Arial", "B", 16)
    pdf.Cell(40, 10, "Fiscal Receipt")
    pdf.Ln(12)
    pdf.SetFont("Arial", "", 12)
    pdf.Cell(40, 10, fmt.Sprintf("User ID: %s", paymentData.UserID))
    pdf.Ln(10)
    pdf.Cell(40, 10, fmt.Sprintf("Name: %s", paymentData.Name))
    pdf.Ln(10)
    pdf.Cell(40, 10, fmt.Sprintf("Address: %s", paymentData.Address))
    pdf.Ln(10)
    pdf.Cell(40, 10, fmt.Sprintf("Card Number: %s", paymentData.CardNumber))
    pdf.Ln(10)
    pdf.Cell(40, 10, fmt.Sprintf("Expiration Date: %s", paymentData.ExpirationDate))
    pdf.Ln(10)
    pdf.Cell(40, 10, fmt.Sprintf("Transaction Date: %s", time.Now().Format("2006-01-02 15:04:05")))

    receiptPath := filepath.Join("receipts", fmt.Sprintf("receipt_%s.pdf", paymentData.UserID))
    err := pdf.OutputFileAndClose(receiptPath)
    if err != nil {
        return "", err
    }
    return receiptPath, nil
}

func subscribeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var paymentData PaymentData
    err := json.NewDecoder(r.Body).Decode(&paymentData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    receiptPath, err := generatePDFReceipt(paymentData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Construct the URL to the generated receipt
    receiptURL := fmt.Sprintf("http://localhost:8081/%s", receiptPath)

    // Return the receipt URL in the response
    response := map[string]string{
        "success":    "true",
        "receiptUrl": receiptURL,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    // Get the absolute path for the payment_form.html
    baseDir, err := os.Getwd()
    if err != nil {
        log.Fatalf("Error getting base directory: %v", err)
    }

    // Create receipts directory if not exists
    receiptsDir := filepath.Join(baseDir, "receipts")
    if _, err := os.Stat(receiptsDir); os.IsNotExist(err) {
        os.Mkdir(receiptsDir, 0755)
    }

    // Serve static files from the 'css', 'images', and 'script' directories
    http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(filepath.Join(baseDir, "css")))))
    http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(filepath.Join(baseDir, "images")))))
    http.Handle("/script/", http.StripPrefix("/script/", http.FileServer(http.Dir(filepath.Join(baseDir, "script")))))
    http.Handle("/receipts/", http.StripPrefix("/receipts/", http.FileServer(http.Dir(receiptsDir))))

    // Handle the payment form route
    http.HandleFunc("/paymentForm", paymentFormHandler)

    // Handle subscription submission
    http.HandleFunc("/subscribe", subscribeHandler)

    // Get the port from environment variable (default to 8081)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"
    }

    // Start the server
    fmt.Printf("Server starting on port %s...\n", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
