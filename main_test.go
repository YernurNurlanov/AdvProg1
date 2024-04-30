package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tebeka/selenium"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Unit test
func TestBsonToDeals(t *testing.T) {
	bsonData := bson.M{
		"_id":     "123456",
		"user_id": "789",
		"card_id": bson.M{
			"card_id":     123,
			"name":        "Test Card",
			"nationality": "Test Nationality",
			"club":        "Test Club",
			"position":    "Test Position",
		},
	}
	deal, err := BsonToDeals(bsonData)
	assert.NoError(t, err, "Unexpected error")

	assert.Equal(t, "123456", deal.ID, "ID mismatch")
	assert.Equal(t, "789", deal.User_id, "User_id mismatch")
	assert.Equal(t, 123, deal.Card_id.Card_id, "Card_id mismatch")
	assert.Equal(t, "Test Card", deal.Card_id.Name, "Name mismatch")
	assert.Equal(t, "Test Nationality", deal.Card_id.Nationality, "Nationality mismatch")
	assert.Equal(t, "Test Club", deal.Card_id.Club, "Club mismatch")
	assert.Equal(t, "Test Position", deal.Card_id.Position, "Position mismatch")
}
// Integration test
func TestRegisterPage(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	registerPage(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Status code mismatch")
}
// End-to-End test
func TestXxx(t *testing.T) {
	clientOptions := options.Client().ApplyURI("mongodb+srv://yernur:1118793421@cluster0.7w9s0t2.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		assert.NoError(t, err, "Unexpected error")
	}
	defer client.Disconnect(context.TODO())
	db = client.Database("Football_Manager")
	service, err := selenium.NewChromeDriverService("./chromedriver", 4444)
  if err != nil {
    t.Fatalf("Failed: %v", err)
  }
	fmt.Println(service)
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, "")
	if err != nil {
		t.Fatalf("Failed to start Selenium session: %v", err)
	}
	// Открытие страницы регистрации
	if err := wd.Get("http://localhost:8080/"); err != nil {
		t.Fatalf("Failed to open registration page: %v", err)
	}
	openButton, err := wd.FindElement(selenium.ByCSSSelector, "button[name='log_in']")
	if err != nil {
			t.Fatalf("Failed to find login button: %v", err)
	}
	openButton.Click()
	// Ввод данных в поля формы и нажатие кнопки
	emailField, err := wd.FindElement(selenium.ByCSSSelector, "input[name='login2']")
	if err != nil {
		t.Fatalf("Failed to find email field: %v", err)
	}
	emailField.SendKeys("admin@gmail.com")

	passwordField, err := wd.FindElement(selenium.ByCSSSelector, "input[name='password2']")
	if err != nil {
		t.Fatalf("Failed to find password field: %v", err)
	}
	passwordField.SendKeys("Admin")
	registerButton, err := wd.FindElement(selenium.ByCSSSelector, "button[name='logbtn']")
	if err != nil {
		t.Fatalf("Failed to find register button: %v", err)
	}
	registerButton.Click()
	time.Sleep(1 * time.Second)
	// Находим кнопку "Add questions" и нажимаем на нее
	for i := 0; i < 100; i++ {
		addQuestionsButton, err := wd.FindElement(selenium.ByCSSSelector, "button.tab-btn:nth-of-type(2)")
		if err != nil {
			t.Fatalf("Failed to find 'Add questions' button: %v", err)
		}
		if err := addQuestionsButton.Click(); err != nil {
			t.Fatalf("Failed to click 'Add questions' button: %v", err)
		}

		

		// Находим поля ввода и заполняем их данными
		questionField, err := wd.FindElement(selenium.ByID, "question")
		if err != nil {
			t.Fatalf("Failed to find question field: %v", err)
		}
		option1Field, err := wd.FindElement(selenium.ByID, "option1")
		if err != nil {
			t.Fatalf("Failed to find option1 field: %v", err)
		}
		// Заполняем поля данными
		if err := questionField.SendKeys("q"+ fmt.Sprint(i)); err != nil {
			t.Fatalf("Failed to fill question field: %v", err)
		}
		if err := option1Field.SendKeys("option1_"+ fmt.Sprint(i)); err != nil {
			t.Fatalf("Failed to fill option1 field: %v", err)
		}
		// Здесь продолжайте заполнять остальные поля
		option2Field, err := wd.FindElement(selenium.ByID, "option2")
		if err != nil {
			t.Fatalf("Failed to find option1 field: %v", err)
		}
		if err := option2Field.SendKeys("option2_"+ fmt.Sprint(i)); err != nil {
			t.Fatalf("Failed to fill option1 field: %v", err)
		}
		option3Field, err := wd.FindElement(selenium.ByID, "option3")
		if err != nil {
			t.Fatalf("Failed to find option1 field: %v", err)
		}
		if err := option3Field.SendKeys("option3_"+ fmt.Sprint(i)); err != nil {
			t.Fatalf("Failed to fill option1 field: %v", err)
		}
		option4Field, err := wd.FindElement(selenium.ByID, "option4")
		if err != nil {
			t.Fatalf("Failed to find option1 field: %v", err)
		}
		if err := option4Field.SendKeys("option4_"+ fmt.Sprint(i)); err != nil {
			t.Fatalf("Failed to fill option1 field: %v", err)
		}
		rightOptionField, err := wd.FindElement(selenium.ByID, "rightOp")
		if err != nil {
			t.Fatalf("Failed to find option1 field: %v", err)
		}
		if err := rightOptionField.SendKeys("a"); err != nil {
			t.Fatalf("Failed to fill option1 field: %v", err)
		}
		// Находим кнопку "Add Question" и нажимаем на нее
		addQuestionButton, err := wd.FindElement(selenium.ByID, "addQuestion")
		if err != nil {
			t.Fatalf("Failed to find 'Add Question' button: %v", err)
		}
		if err := addQuestionButton.Click(); err != nil {
			t.Fatalf("Failed to click 'Add Question' button: %v", err)
		}
		time.Sleep(1 * time.Second)
		if err := wd.AcceptAlert(); err != nil {
			t.Fatalf("Failed to accept alert: %v", err)
		}
		if err := wd.Refresh(); err != nil {
			t.Fatalf("Failed to refresh page: %v", err)
		}
	}
}
func TestGetTeam(t *testing.T) {
	clientOptions := options.Client().ApplyURI("mongodb+srv://yernur:1118793421@cluster0.7w9s0t2.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		assert.NoError(t, err, "Unexpected error")
	}
	defer client.Disconnect(context.TODO())
	db = client.Database("Football_Manager")
	req, err := http.NewRequest("GET", "/getTeam?user_id=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	ctx := context.Background()
	getTeam(rr, req.WithContext(ctx))
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}
	var players []Player
	err = json.Unmarshal(rr.Body.Bytes(), &players)
	if err != nil {
		t.Errorf("error decoding response body: %v", err)
	}
	var expectedPlayers = []Player{
    {"115", "leftForward"},
    {"44", "centerForward"},
    {"40", "rightForward"},
    {"380", "centerMidfielder"},
    {"127", "leftDefender"},
    {"270", "manager"},
	}
	if !reflect.DeepEqual(expectedPlayers, players) {
		t.Errorf("Expected players %v, but got %v", expectedPlayers, players)
	}
}