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
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
	"gopkg.in/gomail.v2"
)

var db *mongo.Database
var logger *logrus.Logger
var tmpl *template.Template
var limiter = rate.NewLimiter(1, 3)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var adminConn *websocket.Conn
var userConns = make(map[*websocket.Conn]bool)
type Chat struct {
	ID string `bson:"_id,omitempty"`
	Id_support string `bson:"id_support"`
	Id_client string `bson:"id_client"`
	Is_finished bool `bson:"is_finished"`
	Messages []Message `bson:"messages"`
}
type Message struct{
		Sender string `bson:"sender_id"`
		Role string `bson:"role"`
		MessageText string `bson:"message"`
		Timestamp   time.Time `bson:"time"`
}
type User struct {
	ID       string `bson:"_id,omitempty"`
	User_id  string `json:"user_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type ConfirmUser struct {
	ID       string `bson:"_id,omitempty"`
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
type newQuestion struct {
	Question string `bson:"question"`
	OptionA  string `bson:"option a"`
	OptionB  string `bson:"option b"`
	OptionC  string `bson:"option c"`
	OptionD  string `bson:"option d"`
	RightOp  string `bson:"right option"`
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
	for i := 0; i < 100; i++ {
		go processQuestions()
	}
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
func createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	email := r.URL.Query().Get("email")
	password := r.URL.Query().Get("password")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		sendErrorMessage(w, "createUser", err, "Error hashing password. Try to register again.")
		return
	}
	newUser := ConfirmUser{
		Email:    email,
		Password: string(hashedPassword),
	}
	existingUser := User{}
	err = db.Collection("users").FindOne(r.Context(), bson.M{"email": email}).Decode(&existingUser)
	if err == nil {
		errorMessage := "User with email: '" + email + "' already exists. Try again."
		sendErrorMessage(w, "createUser", errors.New(errorMessage), errorMessage)
		return
	}
	_, err = db.Collection("confirm_users").InsertOne(r.Context(), newUser)
	if err != nil {
		errorMessage := "error add user. try again"
		sendErrorMessage(w, "createUser", errors.New(errorMessage), errorMessage)
		return
	}
	////////////////////////////////// gmail message
	body := fmt.Sprintf("Для подтверждения почты перейдите по ссылке: https://advprog1.onrender.com/confirmPage?email=%s", email)

	// Настройка клиента для отправки письма
	m := gomail.NewMessage()
	m.SetHeader("From", "nurlanovernur33@gmail.com")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Подтверждение почты")
	m.SetBody("text/plain", body)

	// Отправка письма
	d := gomail.NewDialer("smtp.gmail.com", 587, "nurlanovernur33@gmail.com", "hlwa hlzl epre rhfd")
	if err := d.DialAndSend(m); err != nil {
		sendErrorMessage(w, "createUser", err, "Error send message. Try again.")
		return
	}
	sendSuccessMessage(w, "createUser", "User created successfully. Confirm your email and log in.", "")
}
func confirmPage(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	tmpl, err := template.ParseFiles("confirm.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parsing template file 'collection.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "confirm.html", email)
	logger.WithFields(logrus.Fields{
		"action": "registerPage",
		"status": "success",
	}).Info("User on the page.")
}
func confirm(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	userID, _ := db.Collection("users").CountDocuments(r.Context(), bson.M{})
	var user ConfirmUser
	err := db.Collection("confirm_users").FindOne(r.Context(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error finding user by email. Try again.")
		return
	}
	newUser := User{
		User_id:  strconv.FormatInt(userID+1, 10),
		Email:    user.Email,
		Password: user.Password,
	}
	_, err = db.Collection("users").InsertOne(r.Context(), newUser)
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error adding user. Try again.")
		return
	}
	newUserCollection := Collections{
		User_id: strconv.FormatInt(userID+1, 10),
		Card_id: []Card{},
	}
	_, err = db.Collection("collections").InsertOne(r.Context(), newUserCollection)
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error creating users collection. Try again.")
		return
	}
	questionSize, err := db.Collection("questions").CountDocuments(r.Context(), bson.M{})
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error getting collection size. Try to relod page.")
		return
	}
	randomNumbers := make([]int, 5)
	for i := 0; i < 5; i++ {
		randomNumbers[i] = rand.Intn(int(questionSize))
	}
	cursor, err := db.Collection("questions").Find(r.Context(), bson.M{"question_id": bson.M{"$in": randomNumbers}})
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error adding questions to user. Try to relod page.")
		return
	}
	var questions []Questions
	defer cursor.Close(r.Context())
	for cursor.Next(r.Context()) {
		var question Questions
		err := cursor.Decode(&question)
		if err != nil {
			sendErrorMessage(w, "confirm", err, "Error decoding file. Try to relod page.")
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
		sendErrorMessage(w, "confirm", err, "Error creating users questions. Try again.")
		return
	}
	newUserTeam := Teams{
		User_id:     strconv.FormatInt(userID+1, 10),
		UserPlayers: []Player{},
	}
	_, err = db.Collection("teams").InsertOne(r.Context(), newUserTeam)
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error creating users team. Try again.")
		return
	}
	_, err = db.Collection("confirm_users").DeleteOne(r.Context(), bson.M{"email": email})
	if err != nil {
		sendErrorMessage(w, "confirm", err, "Error creating users team. Try again.")
		return
	}
	sendSuccessMessage(w, "confirm", "Email confirmed successfully. You can log in", "")
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
		sendErrorMessage(w, "login", err, "Incorrect login entered. Try again.")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(password))
	if err != nil {
		sendErrorMessage(w, "login", err, "Incorrect password entered. Try again.")
		return
	}
	token, err := generateToken(&existingUser)

	http.SetCookie(w, &http.Cookie{
			Name:     "auth-token",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "user-data",
		Value: existingUser.ID,
		Path:  "/",
	})
	
	if err != nil {
		sendErrorMessage(w, "login", err, "Failed to generate token. Try to log in again.")
		return
	}
	if email == "admin@gmail.com" {
		sendSuccessMessage(w, "login", "Admin", token)
		return
	}
	sendSuccessMessage(w, "login", "User login successfully", token)
}
func registerPage(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "registration.html", nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	logger.WithFields(logrus.Fields{
		"action": "registerPage",
		"status": "success",
	}).Info("User on the page.")
}
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
	var users []User = getAllUsers(r)
	tmpl.ExecuteTemplate(w, "admin.html", users)
}
func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("id")
	_, err := db.Collection("users").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, "deleteUser", err, "Error deleting user. Try again.")
		return
	}
	_, err = db.Collection("collections").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, "deleteUser", err, "Error deleting users collection. Try again.")
		return
	}
	_, err = db.Collection("user_questions").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, "deleteUser", err, "Error deleting users questions collection. Try again.")
		return
	}
	_, err = db.Collection("teams").DeleteOne(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, "deleteUser", err, "Error deleting users team. Try again.")
		return
	}
	_, err = db.Collection("cards_in_deal").DeleteMany(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		sendErrorMessage(w, "deleteUser", err, "Error deleting users cards in deal. Try again.")
		return
	}
	sendSuccessMessage(w, "deleteUser", "User was deleted.", "")
}
func getAllUsers(r *http.Request) []User {
	var users []User
	cursor, err := db.Collection("users").Find(r.Context(), bson.M{})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "getAllUsers",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error fetching users")
	}
	if err := cursor.All(r.Context(), &users); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "getAllUsers",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error decoding users")
	}
	return users
}
func addCard(w http.ResponseWriter, r *http.Request) {
	type newCard struct {
		Name        string `json:"name"`
		Nationality string `json:"nationality"`
		Club        string `json:"club"`
		Position    string `json:"position"`
	}
	var data newCard
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		sendErrorMessage(w, "addCard", err, "Error decoding new card data. Try again.")
		return
	}
	if data.Position != "Forward" && data.Position != "Midfielder" && data.Position != "Defender" && data.Position != "Manager" && data.Position != "Goalkeeper" {
		sendErrorMessage(w, "addCard", errors.New("invalid input"), "Invalid position. Position must be Forward, Midfielder, Defender, Manager or Goalkeeper.")
		return
	}
	var card Card
	card.Club = data.Club
	card.Name = data.Name
	card.Nationality = data.Nationality
	card.Position = data.Position
	id, err := db.Collection("cards").CountDocuments(r.Context(), bson.M{})
	if err != nil {
		sendErrorMessage(w, "addCard", err, "Error get size of cards collection. Try to reload page.")
		return
	}
	card.Card_id = int(id + 1)
	_, err = db.Collection("cards").InsertOne(r.Context(), card)
	if err != nil {
		sendErrorMessage(w, "addCard", err, "Error add new card to collection. Try again.")
		return
	}
	newsLetter(w, "New card was added: "+card.Name+".")
	sendSuccessMessage(w, "addCard", "New card added to collection successfully!", "")
}
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
		r.URL.Path = "/"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/"
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
	answer := []interface{}{
		tokenString, deals,
	}
	logger.WithFields(logrus.Fields{
		"action": "market",
		"status": "success",
	}).Info("User on the market page")
	tmpl.ExecuteTemplate(w, "market.html", answer)
}
func BsonToDeals(bsonData bson.M) (Deals, error) {
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
			sendErrorMessage(w, "printData", err, "Error decoding database results. Try to reload page.")
			return
		}
		deal, err := BsonToDeals(bsonDoc)
		if err != nil {
			sendErrorMessage(w, "printData", err, "Error converting data to Deals. Try to reload page.")
			return
		}
		deals = append(deals, deal)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deals); err != nil {
		sendErrorMessage(w, "printData", err, "Error encoding JSON. Try to reload page.")
		return
	}
}
func marketCards(w http.ResponseWriter, r *http.Request) {
	var perPage = 20
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		sendErrorMessage(w, "marketCards", err, "Invalid page parameter. Try to reload page.")
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
			sendErrorMessage(w, "marketCards", err, "Error querying the database. Try to reload page.")
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
			sendErrorMessage(w, "marketCards", err, "Error querying the database. Try to reload page.")
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
			sendErrorMessage(w, "marketCards", err, "Error querying the database. Try to reload page.")
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
			sendErrorMessage(w, "marketCards", err, "Error querying the database. Try to reload page.")
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
func cardToCollection(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	cardID := r.URL.Query().Get("card_id")
	card_id, err := strconv.Atoi(cardID)
	if err != nil {
		sendErrorMessage(w, "cardToCollection", err, "Error converting card_id to int. Try again.")
		return
	}
	_, err = db.Collection("cards_in_deal").DeleteOne(r.Context(), bson.M{"user_id": userID, "card_id.card_id": card_id})
	if err != nil {
		sendErrorMessage(w, "cardToCollection", err, "Error deleting card from deal. Try again.")
		return
	}
	var card Card
	err = db.Collection("cards").FindOne(r.Context(), bson.M{"card_id": card_id}, options.FindOne().SetProjection(bson.M{"_id": 0})).Decode(&card)
	if err != nil {
		sendErrorMessage(w, "cardToCollection", err, "Error get card from cards collection. Try again.")
		return
	}
	filter := bson.M{"user_id": userID}
	update := bson.M{"$push": bson.M{"card_id": card}}
	_, err = db.Collection("collections").UpdateOne(r.Context(), filter, update)
	if err != nil {
		sendErrorMessage(w, "cardToCollection", err, "Error put card to your collection. Try again.")
		return
	}
	sendSuccessMessage(w, "cardToCollection", "Player was transfered to collection successfully!", "")
}
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
		r.URL.Path = "/"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "collection",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/"
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
	answer := []interface{}{
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
		sendErrorMessage(w, "saveTeam", err, "Failed to parse request body. Try to reload page.")
		return
	}
	user_id := r.URL.Query().Get("user_id")
	filter := bson.M{"user_id": user_id}
	update := bson.M{"$set": bson.M{"teamPlayers": teamData}}
	// Обновляем документ, соответствующий фильтру
	_, err = db.Collection("teams").UpdateOne(r.Context(), filter, update)
	if err != nil {
		sendErrorMessage(w, "saveTeam", err, "Failed to save team data to MongoDB. Try again.")
		return
	}
	sendSuccessMessage(w, "saveTeam", "Team saved successfully.", "")
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
		sendErrorMessage(w, "getTeam", err, "Failed to fetch team data. Try to reload page.")
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
var mu sync.Mutex
var questionChannel = make(chan newQuestion, 100)
var wg sync.WaitGroup

func processQuestions() {
	for q := range questionChannel {
		var question Questions
		question.Question = q.Question
		question.OptionA = q.OptionA
		question.OptionB = q.OptionB
		question.OptionC = q.OptionC
		question.OptionD = q.OptionD
		question.RightOp = q.RightOp
		id, err := db.Collection("questions").CountDocuments(context.TODO(), bson.M{})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "processQuestions",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error adding question")
			return
		}
		question.QuestionID = int(id + 1)
		_, err = db.Collection("questions").InsertOne(context.TODO(), question)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "processQuestions",
				"status": "error",
				"error":  err.Error(),
			}).Error("Error adding question")
			return
		}
	}
	wg.Done()
}
func addQuestion(w http.ResponseWriter, r *http.Request) {
	var data newQuestion
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		sendErrorMessage(w, "addQuestion", err, "Error decoding new question data. Try again.")
		return
	}
	if data.RightOp != "a" && data.RightOp != "b" && data.RightOp != "c" && data.RightOp != "d" {
		sendErrorMessage(w, "addQuestion", errors.New("invalid input"), "Invalid position. Right option must be 'a', 'b', 'c' or 'd'.")
		return
	}
	mu.Lock()
	defer mu.Unlock()
	questionChannel <- data
	sendSuccessMessage(w, "addQuestion", "New question added to collection successfully!", "")
}
func newsLetter(w http.ResponseWriter, message string) {
	var users []User
	cursor, err := db.Collection("users").Find(context.TODO(), bson.M{})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "newsLetter",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error fetching users")
	}
	if err := cursor.All(context.TODO(), &users); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "newsLetter",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error decoding users")
	}
	for _, user := range users {
		body := fmt.Sprint(message)
		m := gomail.NewMessage()
		m.SetHeader("From", "nurlanovernur33@gmail.com")
		m.SetHeader("To", user.Email)
		m.SetHeader("Subject", "Подтверждение почты")
		m.SetBody("text/plain", body)
		// Отправка письма
		d := gomail.NewDialer("smtp.gmail.com", 587, "nurlanovernur33@gmail.com", "hlwa hlzl epre rhfd")
		if err := d.DialAndSend(m); err != nil {
			sendErrorMessage(w, "createUser", err, "Error send message. Try again.")
			return
		}
	}
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
		r.URL.Path = "/"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "dailyQuestions",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/"
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
	answer := []interface{}{
		tokenString, questions,
	}
	logger.WithFields(logrus.Fields{
		"action": "dailyQuestions",
		"status": "success",
	}).Info("User on the dailyQuestions page")
	tmpl.ExecuteTemplate(w, "daily_card.html", answer)
}

// Chat page
func chatHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("chat.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "chatHandler",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parsing file 'chat.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tokenString := r.URL.Query().Get("token")
	token, err := verifyToken(tokenString)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "chatHandler",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error verifying token")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "chatHandler",
			"status": "error",
			"error":  err,
		}).Error("Failed to convert claims to jwt.Claims.")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "chatHandler",
			"status": "error",
		}).Error("User unauthorized.")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user User
	err = db.Collection("users").FindOne(r.Context(), bson.M{"user_id": userID}, options.FindOne().SetProjection(bson.M{"user_id": 1, "_id": 0})).Decode(&user)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "chatHandler",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error fetching user data.")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		TokenString string
		User        User
	}{
		TokenString: tokenString,
		User:        user,
	}

	logger.WithFields(logrus.Fields{
		"action": "chatHandler",
		"status": "success",
	}).Info("User on the chat page")
	tmpl.ExecuteTemplate(w, "chat.html", data)
}
func allChats(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.Collection("chats").Find(context.TODO(), bson.M{})
	if err != nil {
		sendErrorMessage(w, "allChats", err, "Failed to give all chats. Try again later.")
		return
	}
	var chats []Chat
	for cursor.Next(context.TODO()) {
		var chat Chat
		err := cursor.Decode(&chat)
		if err != nil {
			errorMessage := "Error getting chats"
			logger.WithFields(logrus.Fields{
				"action": "allChats",
				"status": "error",
				"error":  err.Error(),
			}).Error(errorMessage)
			return
		}
		if (!chat.Is_finished && chat.Id_support == "") {
			chats = append(chats, chat)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]Chat{"chats": chats})
}
func myChats(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	cursor, err := db.Collection("chats").Find(context.TODO(), bson.M{"id_support": id})
	if err != nil {
		sendErrorMessage(w, "myChats", err, "Failed to give your chats. Try again later.")
		return
	}
	var chats []Chat
	for cursor.Next(context.TODO()) {
		var chat Chat
		err := cursor.Decode(&chat)
		if err != nil {
			errorMessage := "Error getting chats"
			logger.WithFields(logrus.Fields{
				"action": "myChats",
				"status": "error",
				"error":  err.Error(),
			}).Error(errorMessage)
			return
		}
		if (!chat.Is_finished) {
			chats = append(chats, chat)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]Chat{"chats": chats})
}
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	chatId := r.URL.Query().Get("chat_id")
	objID, err := primitive.ObjectIDFromHex(chatId)
	if err != nil {
			sendErrorMessage(w, "handleAdmin", err, "Invalid chat ID.")
			return
	}
	var chat Chat
	filter := bson.M{"id_support": id, "is_finished": false}
	error := db.Collection("chats").FindOne(r.Context(), filter)
	if error != nil {
		err = db.Collection("chats").FindOne(r.Context(), bson.M{"_id": objID}).Decode(&chat)
		if err != nil {
			sendErrorMessage(w, "handleAdmin", err, "Error getting chat. Try again.")
			return
		}
		chat.Id_support = id
		update := bson.M{"$set": bson.M{"id_support": chat.Id_support}}
		_, err = db.Collection("chats").UpdateOne(r.Context(), bson.M{"_id": objID}, update)
		if err != nil {
			sendErrorMessage(w, "handleAdmin", err, "Error updating chat. Try again.")
			return
		}
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sendErrorMessage(w, "handleAdmin", errors.New("error create connect. try again"), "error create connect. try again")
		return
	}

	mu.Lock()
	adminConn = conn
	mu.Unlock()

	defer func() {
		mu.Lock()
		adminConn = nil
		mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			sendErrorMessage(w, "handleAdmin", errors.New("error read message. try again"), "error read message. try again")
			break
		}
		if string(msg) == "close" {
			mu.Lock()
			for userConn := range userConns {
				if err := userConn.WriteMessage(websocket.TextMessage, []byte("close")); err != nil {
					sendErrorMessage(w, "handleAdmin", errors.New("error write message. try again"), "error write message. try again")
					userConn.Close()
					delete(userConns, userConn)
				}
			}
			mu.Unlock()
			update := bson.M{"$set": bson.M{"is_finished": true}}
			_, err = db.Collection("chats").UpdateOne(r.Context(), bson.M{"_id": objID}, update)
			if err != nil {
				sendErrorMessage(w, "handleAdmin", err, "Error closing chat. Try again.")
				return
			}
			conn.Close()
			break
		}
		message := Message{
			Sender:      id,
			Role: 			 "admin",
			MessageText: string(msg),
			Timestamp:   time.Now(),
		}

		if err := saveMessage(chat.Id_client, message); err != nil {
			conn.Close()
			sendErrorMessage(w, "handleAdmin", err, err.Error())
			break
		}

		mu.Lock()
		for userConn := range userConns {
			if err := userConn.WriteMessage(websocket.TextMessage, msg); err != nil {
				conn.Close()
				sendErrorMessage(w, "handleAdmin", errors.New("error write message. try again"), "error write message. try again")
				userConn.Close()
				delete(userConns, userConn)
			}
		}
		mu.Unlock()
	}
}
func handleUser(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("id")
	filter := bson.M{"id_client": user_id, "is_finished": false}
	var chat Chat
	err := db.Collection("chats").FindOne(r.Context(), filter).Decode(&chat)
	if err != nil && err != mongo.ErrNoDocuments {
			sendErrorMessage(w, "handleUser", err, "Error finding chat")
			return
	}

	if err == mongo.ErrNoDocuments {
		chat := Chat{
		Id_client: user_id,
		Is_finished: false,
		Messages:    []Message{},
		}
		_, err := db.Collection("chats").InsertOne(context.TODO(), chat)
		if err != nil {
			errorMessage := "error create chat. try again"
			sendErrorMessage(w, "handleUser", errors.New(errorMessage), errorMessage)
			return
		}
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sendErrorMessage(w, "handleUser", errors.New("error create connect. try again"), "error create connect. try again")
		return
	}
	mu.Lock()
	userConns[conn] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(userConns, conn)
		mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			sendErrorMessage(w, "handleUser", errors.New("error read message. try again"), "error read message. try again")
			break
		}
		if string(msg) == "close" {
			conn.Close()
			break
		}

		message := Message{
			Sender:      user_id,
			Role: 			 "user",
			MessageText: string(msg),
			Timestamp:   time.Now(),
		}

		if err := saveMessage(user_id, message); err != nil {
			conn.Close()
			sendErrorMessage(w, "handleUser", err, err.Error())
			break
		}

		mu.Lock()
		if adminConn != nil {
			if err := adminConn.WriteMessage(websocket.TextMessage, msg); err != nil {
				conn.Close()
				sendErrorMessage(w, "handleUser", errors.New("error write message. try again"), "error write message. try again")
				mu.Unlock()
				continue
			}
		}
		mu.Unlock()
	}
}
func saveMessage(id_client string, msg Message) error {
	filter := bson.M{"id_client": id_client,"is_finished": false}
	update := bson.M{"$push": bson.M{"messages": msg}}
	result, err := db.Collection("chats").UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("no chat found with id_client: %s", id_client)
	}
	return nil
}
func getChat(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("id")
	role := r.URL.Query().Get("role")
	var chat Chat
	var filter bson.M
	if role == "user" {
		filter = bson.M{"id_client": user_id, "is_finished": false}
	} else {
		filter = bson.M{"id_support": user_id, "is_finished": false}
	}
	err := db.Collection("chats").FindOne(r.Context(), filter).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "No active chat found for the user", http.StatusNotFound)
		} else {
			http.Error(w, "Error finding chat", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(chat.Messages)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
func getToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth-token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": cookie.Value})
}

//
func giveCard(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("user_id")
	answers := r.URL.Query().Get("answers")
	token := r.URL.Query().Get("token")
	if answers == "5" {
		cardSize, err := db.Collection("cards").CountDocuments(r.Context(), bson.M{})
		if err != nil {
			sendErrorMessage(w, "giveCard", err, "Error getting collection size. Try submitting the answers again.")
			return
		}
		randomNumber := rand.Intn(int(cardSize))
		var card Card
		err = db.Collection("cards").FindOne(r.Context(), bson.M{"card_id": randomNumber}, options.FindOne().SetProjection(bson.M{"_id": 0})).Decode(&card)
		if err != nil {
			sendErrorMessage(w, "giveCard", err, "Failed to get the card. Try submitting the answers again.")
			return
		}
		_, err = db.Collection("collections").UpdateOne(r.Context(), bson.M{"user_id": user_id}, bson.M{"$push": bson.M{"card_id": card}})
		if err != nil {
			sendErrorMessage(w, "giveCard", err, "Failed to give the card. Try submitting the answers again.")
			return
		}
		_, err = db.Collection("user_questions").UpdateOne(r.Context(), bson.M{"user_id": user_id}, bson.M{"$set": bson.M{"submit": true}})
		if err != nil {
			sendErrorMessage(w, "giveCard", err, "Failed to submit the answers. Try submitting the answers again.")
			return
		}
		sendSuccessMessage(w, "giveCard", "The card "+card.Name+" has been added to your collection successfully.", token)
	} else {
		_, err := db.Collection("user_questions").UpdateOne(r.Context(), bson.M{"user_id": user_id}, bson.M{"$set": bson.M{"submit": true}})
		if err != nil {
			sendErrorMessage(w, "giveCard", err, "Failed to submit the answers. Try submitting the answers again.")
			return
		}
		sendSuccessMessage(w, "giveCard", "The answers submitted. Good luck getting your new card tomorrow!", token)
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
		}).Error("Error parse file 'homepage.html'")
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
		r.URL.Path = "/"
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/"
		return
	}
	email, ok := claims["email"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/"
		return
	}
	password, ok := claims["password"].(string)
	if !ok {
		logger.WithFields(logrus.Fields{
			"action": "homePage",
			"status": "error",
		}).Error("User unauthorized.")
		r.URL.Path = "/"
		return
	}
	var answer []string
	answer = append(answer, tokenString, userID, email, password)
	tmpl.ExecuteTemplate(w, "homepage.html", answer)
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
}

// Error Handling with log messages
func sendErrorMessage(w http.ResponseWriter, action string, err error, message string) {
	logger.WithFields(logrus.Fields{
		"action": action,
		"status": "error",
		"error":  err.Error(),
	}).Error(message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func sendSuccessMessage(w http.ResponseWriter, action string, message string, token string) {
	logger.WithFields(logrus.Fields{
		"action": action,
		"status": "success",
	}).Info(message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": message, "token": token})
}
// Transaction
type PaymentForm struct {
	UserID         string `json:"user_id" bson:"user_id"`
	CardNumber     string `json:"cardNumber" bson:"cardNumber"`
	ExpirationDate string `json:"expirationDate" bson:"expirationDate"`
	CVV            string `json:"cvv" bson:"cvv"`
	Name           string `json:"name" bson:"name"`
	Address        string `json:"address" bson:"address"`
}

type Transaction struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         string             `json:"user_id" bson:"user_id"`
	CardNumber     string             `json:"cardNumber" bson:"cardNumber"`
	ExpirationDate string             `json:"expirationDate" bson:"expirationDate"`
	CVV            string             `json:"cvv" bson:"cvv"`
	Name           string             `json:"name" bson:"name"`
	Address        string             `json:"address" bson:"address"`
	IsCompleted    bool               `json:"is_completed" bson:"is_completed"`
	Date           time.Time          `json:"date" bson:"date"`
}
func paymentFormHandler(w http.ResponseWriter, r *http.Request){
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
	tmpl.Execute(w, "payment_form.html")
}
func transactionPageHandler(w http.ResponseWriter, r *http.Request){
	tmpl, err := template.ParseFiles("transaction.html")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "transactionPageHandler",
			"status": "error",
			"error":  err.Error(),
		}).Error("Error parsing template file 'transaction.html'")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	logger.WithFields(logrus.Fields{
		"action": "transactionPageHandler",
		"status": "success",
	}).Info("User on the page.")
	tmpl.Execute(w, "transaction.html")
}
func subscribeHandler(w http.ResponseWriter, r *http.Request) {
	var paymentForm PaymentForm
	err := json.NewDecoder(r.Body).Decode(&paymentForm)
	if err != nil {
		sendErrorMessage(w, "subscribeHandler", err, "Error collecting data")
		return
	}
	log.Printf("Received payment form: %+v", paymentForm)
	transaction := Transaction{
		UserID: 				paymentForm.UserID,	
		CardNumber:     paymentForm.CardNumber,
		ExpirationDate: paymentForm.ExpirationDate,
		CVV:            paymentForm.CVV,
		Name:           paymentForm.Name,
		Address:        paymentForm.Address,
		IsCompleted: 		true,
		Date:           time.Now(),
	}
	_, err = db.Collection("transactions").InsertOne(context.TODO(), transaction)
	if err != nil {
		sendErrorMessage(w, "subscribeHandler", err, "Error set data")
		return
	}
	sendSuccessMessage(w, "subscribeHandler", "Data collected successfully, check transactions page, status must be completed", "")
}
func transactions(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("user_id")
	var transaction Transaction
	err := db.Collection("transactions").FindOne(r.Context(), bson.M{"user_id": user_id}).Decode(&transaction)
	if err != nil {
		sendErrorMessage(w, "transactions", err, "Error getting data")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]Transaction{"transaction": transaction})
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
		cookie, err := r.Cookie("auth-token")
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    tokenString := cookie.Value
		
		token, err := verifyToken(tokenString)
		if err != nil || !token.Valid {
			http.Redirect(w, r, "/", http.StatusSeeOther)
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
	// Registration page
	rtr.HandleFunc("/createUser", createUser).Methods("POST")
	rtr.HandleFunc("/login", login).Methods("POST")
	rtr.HandleFunc("/", registerPage).Methods("GET")
	rtr.HandleFunc("/confirmPage", confirmPage)
	rtr.HandleFunc("/confirm", confirm)
	// Market page
	rtr.Handle("/market", authenticate(http.HandlerFunc(market))).Methods("GET")
	rtr.HandleFunc("/marketCards", marketCards).Methods("GET")
	rtr.HandleFunc("/cardToCollection", cardToCollection).Methods("POST")
	// Admin page
	rtr.Handle("/admin", authenticate(http.HandlerFunc(admin)))
	rtr.HandleFunc("/addCard", addCard)
	rtr.HandleFunc("/addQuestion", addQuestion)
	rtr.HandleFunc("/allChats", allChats)
	rtr.HandleFunc("/myChats", myChats)
	rtr.HandleFunc("/deleteUser", deleteUser).Methods("DELETE")
	// Collection page
	rtr.Handle("/collection", authenticate(http.HandlerFunc(collection))).Methods("GET")
	rtr.HandleFunc("/saveTeam", saveTeam)
	rtr.HandleFunc("/getTeam", getTeam)
	rtr.HandleFunc("/get-auth-token", getToken)
	// Daily cards page
	rtr.Handle("/dailyQuestions", authenticate(http.HandlerFunc(dailyQuestions)))
	rtr.HandleFunc("/giveCard", giveCard)
	// Home page
	rtr.Handle("/homePage", authenticate(http.HandlerFunc(homePage))).Methods("GET")
	// Chat Page
	rtr.HandleFunc("/handleAdmin", handleAdmin)
	rtr.HandleFunc("/handleUser", handleUser)
	rtr.HandleFunc("/getChat", getChat)
	rtr.Handle("/chatHandler", authenticate(http.HandlerFunc(chatHandler))).Methods("GET")
	// Transaction
	rtr.HandleFunc("/subscribe", subscribeHandler)
	rtr.Handle("/paymentForm",  authenticate(http.HandlerFunc(paymentFormHandler)))
	rtr.Handle("/transactionPage", authenticate(http.HandlerFunc(transactionPageHandler)))
	rtr.HandleFunc("/transactions", transactions)
	//
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./css"))))
	http.Handle("/script/", http.StripPrefix("/script/", http.FileServer(http.Dir("./script"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
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
