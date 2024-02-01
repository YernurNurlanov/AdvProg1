package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"golang.org/x/time/rate"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database
var logger *logrus.Logger
var tmpl *template.Template
var limiter = rate.NewLimiter(1, 3)

type User struct {
	ID        string    `bson:"_id,omitempty"`
	User_id	  string		`json:"user_id"`
	Login  		string    `json:"login"`
	Password  string    `json:"password"`
}
type Card struct {
	Card_id 	int				`json:"card_id"` 
	Name 			string		`json:"name"`
	Nationality string  `json:"nationality"`
	Club 			string		`json:"club"`
	Position  string		`json:"position"`
}
type Deals struct {
	ID        string    `bson:"_id,omitempty"`
	User_id	  string		`json:"user_id"`
	Card_id  	[]Card   `json:"card_id"`
}
type PageData struct {
	ErrorMsg string
}
func init() {
	logger = logrus.New()

	logger.SetLevel(logrus.DebugLevel)

	logPath := "logs/"
	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create log directory:", err)
	}

	logFileName := filepath.Join(logPath, "app.log")
	logRotation := &lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    10,
		MaxBackups: 30,
		MaxAge:     30,
	}
	tmpl = template.Must(template.ParseFiles("registration.html"))

	logger.SetOutput(logRotation)
}
// CRUD
func getUserByID(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	objID, _ := primitive.ObjectIDFromHex(userID)
	var user User
	err := db.Collection("users").FindOne(r.Context(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "getUserByID",
			"status": "error",
			"error":  err.Error(),
		}).Error("User not found")
		respondWithJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": "User not found",
		})
		return
	}
	logger.WithFields(logrus.Fields{
		"action":     "getUserByID",
		"status":     "success",
		"user":   		user,
	}).Info("User  was found")
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
	objID, _ := primitive.ObjectIDFromHex(updatedUser.ID)
	result, err := db.Collection("users").UpdateOne(r.Context(), bson.M{"_id": objID}, bson.M{"$set": updatedUser})
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("User updated: %d document(s) modified", result.ModifiedCount),
	})
	fmt.Println(updatedUser,objID,updatedUser.ID)
}
func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	objID, _ := primitive.ObjectIDFromHex(userID)

	result, err := db.Collection("users").DeleteOne(r.Context(), bson.M{"_id": objID})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "deleteUser",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error deleting user")
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}
	logger.WithFields(logrus.Fields{
		"action":     "deleteUser",
		"status":     "success",
	}).Info("Users was deleted")
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("User deleted: %d document(s) deleted", result.DeletedCount),
	})
}
func getAllUsers(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.Collection("users").Find(r.Context(), bson.D{})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "getAllUsers",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error fetching users")
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var users []User
	if err := cursor.All(r.Context(), &users); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "getAllUsers",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error decoding users")
		http.Error(w, "Error decoding users", http.StatusInternalServerError)
		return
	}
	logger.WithFields(logrus.Fields{
		"action":     "getAllUsers",
		"status":     "success",
	}).Info("Users was found")
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"users":  users,
	})
}
// Regisration page
func createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "createUser",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parsing form data")
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}
	user_id, _ := db.Collection("users").CountDocuments(r.Context(),bson.M{})
	login := r.FormValue("login")
	password := r.FormValue("password")

	newUser := User{
		User_id: strconv.FormatInt(user_id + 1, 10),
		Login:  login,
		Password:  password,
	}
	existingUser := User{}
	err = db.Collection("users").FindOne(r.Context(), bson.M{"login": login}).Decode(&existingUser)
	if err == nil {
		errorMessage := "User with login: '" + login + "' is already exist."
		logger.WithFields(logrus.Fields{
			"action": "createUser",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
		return
	}
	result, err := db.Collection("users").InsertOne(r.Context(), newUser)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "createUser",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error creating user")
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/market", http.StatusSeeOther)
	logger.WithFields(logrus.Fields{
		"action":     "createUser",
		"status":     "success",
		"insertedID": result.InsertedID,
	}).Info("User created successfully")
}
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "login",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parsing form data")
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}
	login := r.FormValue("login")
	password := r.FormValue("password")
	existingUser := User{}
	err = db.Collection("users").FindOne(r.Context(), bson.M{"login": login,"password": password}).Decode(&existingUser)
	if err != nil {
		errorMessage := "Incorrect login or password entered."
		logger.WithFields(logrus.Fields{
			"action": "login",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
		return
	}
	http.Redirect(w, r, "/market", http.StatusSeeOther)
	logger.WithFields(logrus.Fields{
		"action":     "login",
		"status":     "success",
		"loginID": 		existingUser.ID,
	}).Info("User login successfully")
}
func registerPage(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
	logger.WithFields(logrus.Fields{
		"action":     "registerPage",
		"status":     "success",
	}).Info("User on the page.")
}
// Market page
func market(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("market.html")
	
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "registerPage",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parse file 'market.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	logger.WithFields(logrus.Fields{
		"action":  "registerPage",
		"status":  "success",
	}).Info("User on the market page")
	tmpl.Execute(w, nil)
}
func bsonToDeals(bsonData bson.M) (Deals, error) {
	var deal Deals
	data, err := bson.Marshal(bsonData)
	if err != nil {
		return deal, err
	}
	if err := bson.Unmarshal(data, &deal); err != nil {
		return deal, err
	}
	return deal, nil
}
func printData(w http.ResponseWriter, r *http.Request, cursor *mongo.Cursor) {
	var deals []Deals
	for cursor.Next(r.Context()) {
		var bsonDoc bson.M
		err := cursor.Decode(&bsonDoc)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "printData",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error decoding database results")
			http.Error(w, "Error decoding database results", http.StatusInternalServerError)
			return
		}
		deal, err := bsonToDeals(bsonDoc)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "printData",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error converting to Deals")
			http.Error(w, "Error converting to Deals", http.StatusInternalServerError)
			return
		}
		deals = append(deals, deal)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deals); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "printData",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error encoding JSON")
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
}
func marketCards(w http.ResponseWriter, r *http.Request) {
	var perPage = 20
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "marketCards",
			"status": "error",
			"error":  err.Error(),
		}).Error("Invalid page parameter")
			http.Error(w, "Invalid page parameter", http.StatusBadRequest)
			return
	}
	skip := (page-1) * perPage
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(perPage))

	positionFilter := r.URL.Query().Get("filter")
	sortBy := r.URL.Query().Get("sort")
	if (positionFilter == "" && sortBy == "card_id."){
		cursor, err := db.Collection("cards_in_deals").Find(r.Context(), bson.D{}, findOptions)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "marketCards",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error querying the database")
			http.Error(w, "Error querying the database", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(r.Context())
		printData(w,r,cursor)
		logger.WithFields(logrus.Fields{
			"action":  "marketCards",
			"status":  "success",
			"page":		 page,
		}).Info("Data was printed")
	} else if (positionFilter == "" && sortBy != "card_id.") {
		sortOptions := bson.D{{Key: sortBy, Value: 1}}
		findOptions.SetSort(sortOptions)
		cursor, err := db.Collection("cards_in_deals").Find(r.Context(), bson.M{}, findOptions)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "marketCards",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error querying the database")
			http.Error(w, "Error querying the database", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(r.Context())
		printData(w,r,cursor)
		logger.WithFields(logrus.Fields{
			"action":  "marketCards",
			"status":  "success",
			"page":		 page,
			"sortBy":	 sortBy,	
		}).Info("Data was sorted by " + sortBy)
	} else if (positionFilter != "" && sortBy == "card_id."){
		filter := bson.M{"card_id.position": positionFilter}
		cursor, err := db.Collection("cards_in_deals").Find(r.Context(), filter, findOptions)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "marketCards",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error querying the database")
			http.Error(w, "Error querying the database", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(r.Context())
		printData(w,r,cursor)
		logger.WithFields(logrus.Fields{
			"action":  "marketCards",
			"status":  "success",
			"page":		 page,
			"filter":	 positionFilter,
		}).Info("Data was filtered by " + positionFilter)
	} else {
		sortOptions := bson.D{{Key: sortBy, Value: 1}}
		findOptions.SetSort(sortOptions)
		filter := bson.M{"card_id.position": positionFilter}
		cursor, err := db.Collection("cards_in_deals").Find(r.Context(), filter, findOptions)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "marketCards",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error querying the database")
			http.Error(w, "Error querying the database", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(r.Context())
		printData(w,r,cursor)
		logger.WithFields(logrus.Fields{
			"action":  "marketCards",
			"status":  "success",
			"page":		 page,
			"sortBy":	 sortBy,
			"filter":	 positionFilter,
		}).Info("Data was sorted by " + sortBy + " and filtered by " + positionFilter)
	}
}
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
func handler(w http.ResponseWriter, r *http.Request){
	if !limiter.Allow() {
		<-time.Tick(1 * time.Second)
	}
}
func handleRequests() {
	rtr := mux.NewRouter()
	//  CRUD
	rtr.HandleFunc("/getUserByID", getUserByID).Methods("GET")
	rtr.HandleFunc("/updateUser", updateUser).Methods("PUT","POST")
	rtr.HandleFunc("/deleteUser", deleteUser).Methods("DELETE")
	rtr.HandleFunc("/getAllUsers", getAllUsers).Methods("GET")
	// Registration page
	rtr.HandleFunc("/createUser", createUser).Methods("POST")
	rtr.HandleFunc("/login", login).Methods("POST")
	rtr.HandleFunc("/register", registerPage).Methods("GET")
	// Market page
	rtr.HandleFunc("/market", market).Methods("GET")
	rtr.HandleFunc("/marketCards", marketCards).Methods("GET")
	//
	rtr.HandleFunc("/", handler)
	http.Handle("/", rtr)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
func main() {
	// connection with db
	clientOptions := options.Client().ApplyURI("mongodb://127.0.0.1:27017/")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())
	db = client.Database("football_tools")
	// 
	handleRequests()
}