package main
import (
  "encoding/json"
  "fmt"
  "net/http"
)
type JsonRequest struct {
	Message string `json:"message"`
}
type JsonResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}
		var request JsonRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&request); err != nil {
			jsonResponse := JsonResponse{
        Status:  "400",
        Message: "Invalid JSON message",
			}
    	respondWithJSON(w, http.StatusBadRequest, jsonResponse)
    	return
		}
		if request.Message != "Hello, server! This is JSON data from Postman" {
			jsonResponse := JsonResponse{
        Status:  "400",
        Message: "Invalid JSON message",
			}
    	respondWithJSON(w, http.StatusBadRequest, jsonResponse)
    	return
		}
		fmt.Printf("Received message: %s\n", request.Message)
		response := JsonResponse{
			Status:  "success",
			Message: "Data successfully received",
		}
		respondWithJSON(w, http.StatusOK, response)
	})
	fmt.Println("Port 8080")
	http.ListenAndServe(":8080", nil)
}