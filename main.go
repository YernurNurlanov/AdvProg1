package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database

type User struct {
	ID        string    `bson:"_id,omitempty"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	email := r.FormValue("email")

	newUser := User{
		Username:  username,
		Password:  password,
		Email:     email,
		CreatedAt: time.Now(),
	}

	result, err := db.Collection("users").InsertOne(r.Context(), newUser)
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "User successfully created",
		"user_id": result.InsertedID,
	})
}


func getUserByID(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")

	var user User
	err := db.Collection("users").FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		respondWithJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": "User not found",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"user":   user,
	})
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	var updatedUser User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&updatedUser); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	result, err := db.Collection("users").UpdateOne(r.Context(), bson.M{"_id": updatedUser.ID}, bson.M{"$set": updatedUser})
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("User updated: %d document(s) modified", result.ModifiedCount),
	})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")

	result, err := db.Collection("users").DeleteOne(r.Context(), bson.M{"_id": userID})
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("User deleted: %d document(s) deleted", result.DeletedCount),
	})
}

func getAllUsers(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.Collection("users").Find(r.Context(), bson.D{})
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var users []User
	if err := cursor.All(r.Context(), &users); err != nil {
		log.Fatal(err)
		http.Error(w, "Error decoding users", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"users":  users,
	})
}

func registerPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("register").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Registration Page</title>
		</head>
		<body>
			<h1>Registration</h1>
			<form action="/createUser" method="post">
				<label for="username">Username:</label>
				<input type="text" id="username" name="username" required><br>

				<label for="password">Password:</label>
				<input type="password" id="password" name="password" required><br>

				<label for="email">Email:</label>
				<input type="email" id="email" name="email" required><br>

				<input type="submit" value="Register">
			</form>
		</body>
		</html>
	`)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func handleRequests() {
	http.HandleFunc("/createUser", createUser)
	http.HandleFunc("/getUserByID", getUserByID)
	http.HandleFunc("/updateUser", updateUser)
	http.HandleFunc("/deleteUser", deleteUser)
	http.HandleFunc("/getAllUsers", getAllUsers)
	http.HandleFunc("/register", registerPage)

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}

func main() {
	clientOptions := options.Client().ApplyURI("mongodb://127.0.0.1:27017/")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	db = client.Database("advprog1")

	handleRequests()
}