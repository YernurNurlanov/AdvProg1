package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var db *mongo.Database
var logger *logrus.Logger
var tmpl *template.Template
var limiter = rate.NewLimiter(1, 3)

type User struct {
	ID       string `bson:"_id,omitempty"`
	User_id  string `json:"user_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type Card struct {
	Card_id     int    `json:"card_id"`
	Name        string `json:"name"`
	Nationality string `json:"nationality"`
	Club        string `json:"club"`
	Position    string `json:"position"`
}
type Deals struct {
	ID      string `bson:"_id,omitempty"`
	User_id string `json:"user_id"`
	Card_id Card   `json:"card_id"`
}
type Collections struct {
	User_id string `json:"user_id"`
	Card_id []Card `json:"card_id"`
}
type Card_in_deal struct {
	User_id string `json:"user_id"`
	Card_id Card   `json:"card_id"`
}
type User_Questions struct {
	User_id       string      `json:"user_id"`
	UserQuestions []Questions `bson:"questions"`
	Submit        bool        `json:"submit"`
}
type Questions struct {
	ID         string `bson:"_id"`
	QuestionID int    `bson:"question_id"`
	Question   string `bson:"question"`
	OptionA    string `bson:"option a"`
	OptionB    string `bson:"option b"`
	OptionC    string `bson:"option c"`
	OptionD    string `bson:"option d"`
	RightOp    string `bson:"right option"`
}
type Teams struct {
	User_id     string   `json:"user_id"`
	UserPlayers []Player `bson:"teamPlayers"`
}
type Player struct {
	PlayerID string `json:"playerId"`
	Position string `json:"position"`
}

// Log file
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
		"action": "getUserByID",
		"status": "success",
		"user":   user,
	}).Info("User  was found")
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"user":   user,
	})
}

// НАДО ИСПРАВИТЬ
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
}
func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("id")
	_, err := db.Collection("users").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, r, "deleteUser", err, "Error deleting user. Try again.")
		return
	}
	_, err = db.Collection("collections").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, r, "deleteUser", err, "Error deleting users collection. Try again.")
		return
	}
	_, err = db.Collection("user_questions").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, r, "deleteUser", err, "Error deleting users questions collection. Try again.")
		return
	}
	_, err = db.Collection("teams").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, r, "deleteUser", err, "Error deleting users team. Try again.")
		return
	}
	_, err = db.Collection("cards_in_deal").DeleteMany(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, r, "deleteUser", err, "Error deleting users cards in deal. Try again.")
		return
	}
	sendSuccessMessage(w, r, "deleteUser", "User was deleted.", "")
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
		"action": "getAllUsers",
		"status": "success",
	}).Info("Users was found")
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"users":  users,
	})
}

// /////////////////////////////////////////////////////////////////// Regisration page
func createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	// НАДО ИСПРАВИТЬ
	userID, _ := db.Collection("users").CountDocuments(r.Context(), bson.M{})
	email := r.URL.Query().Get("email")
	password := r.URL.Query().Get("password")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		sendErrorMessage(w, r, "createUser", err, "Error hashing password. Try to register again.")
		return
	}
	newUser := User{
		User_id:  strconv.FormatInt(userID+1, 10),
		Email:    email,
		Password: string(hashedPassword),
	}
	existingUser := User{}
	err = db.Collection("users").FindOne(r.Context(), bson.M{"email": email}).Decode(&existingUser)
	if err == nil {
		errorMessage := "User with email: '" + email + "' already exists. Try again."
		sendErrorMessage(w, r, "createUser", errors.New(errorMessage), errorMessage)
		return
	}
	_, err = db.Collection("users").InsertOne(r.Context(), newUser)
	if err != nil {
		sendErrorMessage(w, r, "createUser", err, "Error creating user. Try again.")
		return
	}
	newUserCollection := Collections{
		User_id: strconv.FormatInt(userID+1, 10),
		Card_id: []Card{},
	}
	_, err = db.Collection("collections").InsertOne(r.Context(), newUserCollection)
	if err != nil {
		sendErrorMessage(w, r, "createUser", err, "Error creating users collection. Try again.")
		return
	}
	questionSize, err := db.Collection("questions").CountDocuments(r.Context(), bson.M{})
	if err != nil {
		errorMessage := "Error getting collection size"
		logger.WithFields(logrus.Fields{
			"action": "createUser",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
		return
	}
	randomNumbers := make([]int, 5)
	for i := 0; i < 5; i++ {
		randomNumbers[i] = rand.Intn(int(questionSize))
	}
	cursor, err := db.Collection("questions").Find(r.Context(), bson.M{"question_id": bson.M{"$in": randomNumbers}})
	if err != nil {
		errorMessage := "User has not players in collection."
		logger.WithFields(logrus.Fields{
			"action": "createUser",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
		return
	}
	var questions []Questions
	defer cursor.Close(r.Context())
	for cursor.Next(r.Context()) {
		var question Questions
		err := cursor.Decode(&question)
		if err != nil {
			errorMessage := "Error decoding file"
			logger.WithFields(logrus.Fields{
				"action": "updateCollectionPeriodically",
				"status": "error",
				"error":  err.Error(),
			}).Error(errorMessage)
			return
		}
		questions = append(questions, question)
	}
	newUserQuestions := User_Questions{
		User_id:       strconv.FormatInt(userID+1, 10),
		UserQuestions: questions,
		Submit:        false,
	}
	_, err = db.Collection("user_questions").InsertOne(r.Context(), newUserQuestions)
	if err != nil {
		sendErrorMessage(w, r, "createUser", err, "Error creating users questions. Try again.")
		return
	}
	newUserTeam := Teams{
		User_id:     strconv.FormatInt(userID+1, 10),
		UserPlayers: []Player{},
	}
	_, err = db.Collection("teams").InsertOne(r.Context(), newUserTeam)
	if err != nil {
		sendErrorMessage(w, r, "createUser", err, "Error creating users team. Try again.")
		return
	}
	sendSuccessMessage(w, r, "createUser", "User created successfully. Log in to the account", "")
}
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	email := r.URL.Query().Get("email")
	password := r.URL.Query().Get("password")
	existingUser := User{}
	err := db.Collection("users").FindOne(r.Context(), bson.M{"email": email}).Decode(&existingUser)
	if err != nil {
		sendErrorMessage(w, r, "login", err, "Incorrect login entered. Try again.")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(password))
	if err != nil {
		sendErrorMessage(w, r, "login", err, "Incorrect password entered. Try again.")
		return
	}
	token, err := generateToken(&existingUser)
	if err != nil {
		sendErrorMessage(w, r, "login", err, "Failed to generate token. Try to log in again.")
		return
	}
	if email == "admin@gmail.com" {
		sendSuccessMessage(w, r, "login", "Admin", token)
		return
	}
	sendSuccessMessage(w, r, "login", "User login successfully", token)
}
func registerPage(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "registration.html", nil)
	logger.WithFields(logrus.Fields{
		"action": "registerPage",
		"status": "success",
	}).Info("User on the page.")
}

// //////////////////////////////////////////////////////////////////// Market page
func market(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	token, err := verifyToken(tokenString)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "market",
			"status": "error",
			"error":  err,
		}).Error("Failed to convert claims to jwt.Claims.")
		r.URL.Path = "/register"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/register"
		return
	}
	tmpl, err := template.ParseFiles("market.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "market",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parse file 'market.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	var deals []Card_in_deal
	cursor, err := db.Collection("cards_in_deal").Find(r.Context(), bson.M{"user_id": userID}, options.Find().SetProjection(bson.M{"user_id": 1, "card_id": 1, "_id": 0}))
	if err != nil {
		errorMessage := "User has not players in deals."
		logger.WithFields(logrus.Fields{
			"action": "market",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
	}
	defer cursor.Close(r.Context())
	for cursor.Next(r.Context()) {
		var deal Card_in_deal
		err := cursor.Decode(&deal)
		if err != nil {
			errorMessage := "Error decoding file"
			logger.WithFields(logrus.Fields{
				"action": "dailyQuestions",
				"status": "error",
				"error":  err.Error(),
			}).Error(errorMessage)
		}
		deals = append(deals, deal)
	}
	answer := []interface{} {
			tokenString, deals,
		}
	logger.WithFields(logrus.Fields{
		"action": "market",
		"status": "success",
	}).Info("User on the market page")
	tmpl.ExecuteTemplate(w, "market.html", answer)
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
			sendErrorMessage(w, r, "printData", err, "Error decoding database results. Try to reload page.")
			return
		}
		deal, err := bsonToDeals(bsonDoc)
		if err != nil {
			sendErrorMessage(w, r, "printData", err, "Error converting data to Deals. Try to reload page.")
			return
		}
		deals = append(deals, deal)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deals); err != nil {
		sendErrorMessage(w, r, "printData", err, "Error encoding JSON. Try to reload page.")
		return
	}
}
func marketCards(w http.ResponseWriter, r *http.Request) {
	var perPage = 20
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		sendErrorMessage(w, r, "marketCards", err, "Invalid page parameter. Try to reload page.")
		return
	}
	skip := (page - 1) * perPage
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(perPage))
	positionFilter := r.URL.Query().Get("filter")
	sortBy := r.URL.Query().Get("sort")
	if positionFilter == "" && sortBy == "card_id." {
		cursor, err := db.Collection("cards_in_deal").Find(r.Context(), bson.D{}, findOptions)
		if err != nil {
			sendErrorMessage(w, r, "marketCards", err, "Error querying the database. Try to reload page.")
			return
		}
		defer cursor.Close(r.Context())
		printData(w, r, cursor)
		logger.WithFields(logrus.Fields{
			"action": "marketCards",
			"status": "success",
			"page":   page,
		}).Info("Data was printed")
	} else if positionFilter == "" && sortBy != "card_id." {
		sortOptions := bson.D{{Key: sortBy, Value: 1}}
		findOptions.SetSort(sortOptions)
		cursor, err := db.Collection("cards_in_deal").Find(r.Context(), bson.M{}, findOptions)
		if err != nil {
			sendErrorMessage(w, r, "marketCards", err, "Error querying the database. Try to reload page.")
			return
		}
		defer cursor.Close(r.Context())
		printData(w, r, cursor)
		logger.WithFields(logrus.Fields{
			"action": "marketCards",
			"status": "success",
			"page":   page,
			"sortBy": sortBy,
		}).Info("Data was sorted by " + sortBy)
	} else if positionFilter != "" && sortBy == "card_id." {
		filter := bson.M{"card_id.position": positionFilter}
		cursor, err := db.Collection("cards_in_deal").Find(r.Context(), filter, findOptions)
		if err != nil {
			sendErrorMessage(w, r, "marketCards", err, "Error querying the database. Try to reload page.")
			return
		}
		defer cursor.Close(r.Context())
		printData(w, r, cursor)
		logger.WithFields(logrus.Fields{
			"action": "marketCards",
			"status": "success",
			"page":   page,
			"filter": positionFilter,
		}).Info("Data was filtered by " + positionFilter)
	} else {
		sortOptions := bson.D{{Key: sortBy, Value: 1}}
		findOptions.SetSort(sortOptions)
		filter := bson.M{"card_id.position": positionFilter}
		cursor, err := db.Collection("cards_in_deal").Find(r.Context(), filter, findOptions)
		if err != nil {
			sendErrorMessage(w, r, "marketCards", err, "Error querying the database. Try to reload page.")
			return
		}
		defer cursor.Close(r.Context())
		printData(w, r, cursor)
		logger.WithFields(logrus.Fields{
			"action": "marketCards",
			"status": "success",
			"page":   page,
			"sortBy": sortBy,
			"filter": positionFilter,
		}).Info("Data was sorted by " + sortBy + " and filtered by " + positionFilter)
	}
}

// НАДО ИСПРАВИТЬ
func cardToCollection(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	cardID := r.URL.Query().Get("card_id")
	_, err := db.Collection("cards_in_deal").DeleteOne(r.Context(), bson.M{"user_id": userID, "card_id": cardID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var card Card
	err = db.Collection("cards").FindOne(r.Context(), bson.M{"card_id": cardID}, options.FindOne().SetProjection(bson.M{"_id": 0})).Decode(&card)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filter := bson.M{"user_id": userID}
	update := bson.M{"$push": bson.M{"cards": card}}
	_, err = db.Collection("collections").UpdateOne(r.Context(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// /////////////////////////////////////////////////////////////////// Collection page
func collection(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	token, err := verifyToken(tokenString)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
			"error":  err,
		}).Error("Failed to convert claims to jwt.Claims.")
		r.URL.Path = "/register"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/register"
		return
	}
	tmpl, err := template.ParseFiles("collection.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parsing template file 'collection.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	var collection Collections
	err = db.Collection("collections").FindOne(r.Context(), bson.M{"user_id": userID}, options.FindOne().SetProjection(bson.M{"user_id": 1, "card_id": 1, "_id": 0})).Decode(&collection)
	if err != nil {
		errorMessage := "User has no players in collection."
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
	}
	answer := []interface{} {
		tokenString, collection,
	}
	logger.WithFields(logrus.Fields{
		"action": "collection",
		"status": "success",
		"userID": userID,
	}).Info("User accessed the collection page")
	tmpl.ExecuteTemplate(w, "collection.html", answer)
}
func saveTeam(w http.ResponseWriter, r *http.Request) {
	var teamData []Player
	err := json.NewDecoder(r.Body).Decode(&teamData)
	if err != nil {
		sendErrorMessage(w, r, "saveTeam", err, "Failed to parse request body. Try to reload page.")
		return
	}
	user_id := r.URL.Query().Get("user_id")
	filter := bson.M{"user_id": user_id}
	update := bson.M{"$set": bson.M{"teamPlayers": teamData}}
	// Обновляем документ, соответствующий фильтру
	_, err = db.Collection("teams").UpdateOne(r.Context(), filter, update)
	if err != nil {
		sendErrorMessage(w, r, "saveTeam", err, "Failed to save team data to MongoDB. Try again.")
		return
	}
	sendSuccessMessage(w, r, "saveTeam", "Team saved successfully.", "")
}
func getTeam(w http.ResponseWriter, r *http.Request) {
	type TeamPlayers struct {
		Players []Player `bson:"teamPlayers"`
	}
	user_id := r.URL.Query().Get("user_id")
	var teamPlayers TeamPlayers
	filter := bson.M{"user_id": user_id}
	err := db.Collection("teams").FindOne(r.Context(), filter).Decode(&teamPlayers)
	if err != nil {
		sendErrorMessage(w, r, "getTeam", err, "Failed to fetch team data. Try to reload page.")
		return
	}
	var players []Player
	for _, playerInfo := range teamPlayers.Players {
		players = append(players, Player{
			PlayerID: playerInfo.PlayerID,
			Position: playerInfo.Position,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

// /////////////////////////////////////////////////////////////////// Admin page
func admin(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("admin.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "admin",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parse file 'admin.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	logger.WithFields(logrus.Fields{
		"action": "admin",
		"status": "success",
	}).Info("User on the admin page")
	var users []User
	cursor, err := db.Collection("users").Find(r.Context(), bson.M{})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "admin",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error fetching users")
	}
	if err := cursor.All(r.Context(), &users); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "admin",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error decoding users")
	}
	tmpl.ExecuteTemplate(w, "admin.html", users)
}

// /////////////////////////////////////////////////////////////////// Daily questions page
func dailyQuestions(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("daily_card.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "dailyQuestions",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parse file 'daily_card.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tokenString := r.URL.Query().Get("token")
	token, err := verifyToken(tokenString)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "dailyQuestions",
			"status": "error",
			"error":  err,
		}).Error("Failed to convert claims to jwt.Claims.")
		r.URL.Path = "/register"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "dailyQuestions",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/register"
		return
	}
	var questions User_Questions
	err = db.Collection("user_questions").FindOne(r.Context(), bson.M{"user_id": userID}, options.FindOne().SetProjection(bson.M{"user_id": 1, "questions": 1, "submit": 1, "_id": 0})).Decode(&questions)
	if err != nil {
		errorMessage := "User has no questions."
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
			"error":  err.Error(),
		}).Error(errorMessage)
	}
	answer := []interface{} {
		tokenString, questions,
	}
	logger.WithFields(logrus.Fields{
		"action": "dailyQuestions",
		"status": "success",
	}).Info("User on the dailyQuestions page")
	tmpl.ExecuteTemplate(w, "daily_card.html", answer)
}
func giveCard(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("user_id")
	answers := r.URL.Query().Get("answers")
	if answers == "5" {
		cardSize, err := db.Collection("cards").CountDocuments(r.Context(), bson.M{})
		if err != nil {
			sendErrorMessage(w, r, "giveCard", err, "Error getting collection size. Try submitting the answers again.")
			return
		}
		randomNumber := rand.Intn(int(cardSize))
		var card Card
		err = db.Collection("cards").FindOne(r.Context(), bson.M{"card_id": randomNumber}, options.FindOne().SetProjection(bson.M{"_id": 0})).Decode(&card)
		if err != nil {
			sendErrorMessage(w, r, "giveCard", err, "Failed to get the card. Try submitting the answers again.")
			return
		}
		_, err = db.Collection("collections").UpdateOne(r.Context(), bson.M{"user_id": user_id}, bson.M{"$push": bson.M{"card_id": card}})
		if err != nil {
			sendErrorMessage(w, r, "giveCard", err, "Failed to give the card. Try submitting the answers again.")
			return
		}
		_, err = db.Collection("user_questions").UpdateOne(r.Context(), bson.M{"user_id": user_id}, bson.M{"$set": bson.M{"submit": true}})
		if err != nil {
			sendErrorMessage(w, r, "giveCard", err, "Failed to submit the answers. Try submitting the answers again.")
			return
		}
		sendSuccessMessage(w, r, "giveCard", "The card "+card.Name+" has been added to your collection successfully.", "")
	} else {
		_, err := db.Collection("user_questions").UpdateOne(r.Context(), bson.M{"user_id": user_id}, bson.M{"$set": bson.M{"submit": true}})
		if err != nil {
			sendErrorMessage(w, r, "giveCard", err, "Failed to submit the answers. Try submitting the answers again.")
			return
		}
		sendSuccessMessage(w, r, "giveCard", "The answers submitted. Good luck getting your new card tomorrow!", "")
	}
}
func updateCollectionPeriodically(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		questionSize, err := db.Collection("questions").CountDocuments(ctx, bson.M{})
		if err != nil {
			errorMessage := "Error getting collection size"
			logger.WithFields(logrus.Fields{
				"action": "updateCollectionPeriodically",
				"status": "error",
				"error":  err.Error(),
			}).Error(errorMessage)
			return
		}
		collectionSize, err := db.Collection("user_questions").CountDocuments(ctx, bson.M{})
		if err != nil {
			errorMessage := "Error getting collection size"
			logger.WithFields(logrus.Fields{
				"action": "updateCollectionPeriodically",
				"status": "error",
				"error":  err.Error(),
			}).Error(errorMessage)
			return
		}
		for userID := 1; userID < int(collectionSize); userID++ {
			randomNumbers := make([]int, 5)
			for i := 0; i < 5; i++ {
				randomNumbers[i] = rand.Intn(int(questionSize))
			}
			cursor, err := db.Collection("questions").Find(ctx, bson.M{"question_id": bson.M{"$in": randomNumbers}})
			if err != nil {
				errorMessage := "User has not players in collection."
				logger.WithFields(logrus.Fields{
					"action": "updateCollectionPeriodically",
					"status": "error",
					"error":  err.Error(),
				}).Error(errorMessage)
				return
			}
			var questions []Questions
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var question Questions
				err := cursor.Decode(&question)
				if err != nil {
					errorMessage := "Error decoding file"
					logger.WithFields(logrus.Fields{
						"action": "updateCollectionPeriodically",
						"status": "error",
						"error":  err.Error(),
					}).Error(errorMessage)
					return
				}
				questions = append(questions, question)
			}
			filter := bson.M{"user_id": strconv.Itoa(userID)}
			update := bson.M{"$set": bson.M{"questions": questions, "submit": false}}
			_, err = db.Collection("user_questions").UpdateOne(ctx, filter, update)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"action": "updateCollectionPeriodically",
					"status": "error",
					"error":  err.Error(),
				}).Error("Failed to save team data to MongoDB.")
				return
			}
		}
	}
}
// /////////////////////////////////////////////////////////////////// Home page
func homePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("homepage.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parse file 'daily_card.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tokenString := r.URL.Query().Get("token")
	token, err := verifyToken(tokenString)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
			"error":  err,
		}).Error("Failed to convert claims to jwt.Claims.")
		r.URL.Path = "/register"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/register"
		return
	}
	email, ok := claims["email"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/register"
		return
	}
	password, ok := claims["password"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/register"
		return
	}
	var answer []string
	answer = append(answer, tokenString, userID, email, password)
	tmpl.ExecuteTemplate(w, "homepage.html", answer)
}
// Error Handling with log messages
func sendErrorMessage(w http.ResponseWriter, r *http.Request, action string, err error, message string) {
	logger.WithFields(logrus.Fields{
		"action": action,
		"status": "error",
		"error":  err.Error(),
	}).Error(message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func sendSuccessMessage(w http.ResponseWriter, r *http.Request, action string, message string, token string) {
	logger.WithFields(logrus.Fields{
		"action": action,
		"status": "success",
	}).Info(message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": message, "token": token})
}
// JWT token
func generateToken(user *User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := jwt.MapClaims{}
	claims["user_id"] = user.User_id
	claims["email"] = user.Email
	claims["password"] = user.Password
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	token.Claims = claims

	tokenString, err := token.SignedString([]byte("secret_key"))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
func verifyToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret_key"), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}
func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.URL.Query().Get("token")
		if tokenString == "" {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}
		token, err := verifyToken(tokenString)
		if err != nil || !token.Valid {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
func handler(w http.ResponseWriter, r *http.Request) {
	if !limiter.Allow() {
		<-time.Tick(1 * time.Second)
	}
}
func handleRequests() {
	rtr := mux.NewRouter()
	//  CRUD
	rtr.HandleFunc("/getUserByID", getUserByID).Methods("GET")
	rtr.HandleFunc("/updateUser", updateUser).Methods("PUT", "POST")
	rtr.HandleFunc("/deleteUser", deleteUser).Methods("DELETE")
	rtr.HandleFunc("/getAllUsers", getAllUsers).Methods("GET")
	// Registration page
	rtr.HandleFunc("/createUser", createUser).Methods("POST")
	rtr.HandleFunc("/login", login).Methods("POST")
	rtr.HandleFunc("/register", registerPage).Methods("GET")
	// Market page
	rtr.Handle("/market", authenticate(http.HandlerFunc(market))).Methods("GET")
	rtr.HandleFunc("/marketCards", marketCards).Methods("GET")
	rtr.HandleFunc("/cardToCollection", cardToCollection)
	// Admin page
	rtr.Handle("/admin", authenticate(http.HandlerFunc(admin)))
	// Collection page
	rtr.Handle("/collection", authenticate(http.HandlerFunc(collection))).Methods("GET")
	rtr.HandleFunc("/saveTeam", saveTeam)
	rtr.HandleFunc("/getTeam", getTeam)
	// Daily cards page
	rtr.Handle("/dailyQuestions", authenticate(http.HandlerFunc(dailyQuestions)))
	rtr.HandleFunc("/giveCard", giveCard)
	// Home page
	rtr.Handle("/homePage", authenticate(http.HandlerFunc(homePage)))
	rtr.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("images"))))
	//
	rtr.HandleFunc("/", handler)
	http.Handle("/", rtr)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
func main() {
	// connection with db
	clientOptions := options.Client().ApplyURI("mongodb+srv://yernur:1118793421@cluster0.7w9s0t2.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())
	db = client.Database("Football_Manager")
	//
	ctx := context.Background()
	go updateCollectionPeriodically(ctx)
	handleRequests()
}
