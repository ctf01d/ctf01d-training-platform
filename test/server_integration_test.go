package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ctf01d/config"
	"ctf01d/internal/app/handlers"
	migration "ctf01d/internal/app/migrations/psql"
	"ctf01d/internal/app/server"

	"github.com/jaswdr/faker"
	_ "github.com/lib/pq"

	"github.com/go-chi/chi/v5"
)

var db *sql.DB

func TestMain(m *testing.M) {
	cfg, err := config.NewConfig("../config/config.test.yml")
	if err != nil {
		panic(err)
	}

	db, err = migration.InitDatabase(cfg)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	db.Close()
	os.Exit(code)
}

func NewTestRouter() (http.Handler, error) {
	hndlrs := &handlers.Handlers{
		DB: db,
	}
	svr := handlers.NewServerInterfaceWrapper(hndlrs)

	r := chi.NewRouter()
	r.Mount("/api/", server.HandlerFromMux(svr, r))
	r.Mount("/", http.HandlerFunc(server.NewHtmlRouter))

	return r, nil
}

func TestUserCRUD(t *testing.T) {
	r, err := NewTestRouter()
	if err != nil {
		t.Fatalf("failed to initialize router: %v", err)
	}

	var userID string
	var createdUser map[string]interface{}
	fake := faker.New()

	// 1. Создание пользователя
	t.Run("Create User", func(t *testing.T) {
		createdUser = map[string]interface{}{
			"display_name": fake.Person().Name(),
			"user_name":    fake.Gamer().Tag(),
			"role":         "player",
			"avatar_url":   "http://example.com/avatar.png",
			"status":       "active",
			"password":     fake.Internet().Password(),
		}
		body, _ := json.Marshal(createdUser)
		req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		userID = response["id"].(string)
		if userID == "" {
			t.Fatalf("expected user ID in response")
		}

		delete(createdUser, "password") // Убираем поле с паролем
		// Проверка всех полей ответа
		for key, value := range createdUser {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 2. Получение всех пользователей и использование ID последнего
	t.Run("Get All Users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var users []map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &users); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if len(users) == 0 {
			t.Fatalf("expected at least one user")
		}

		lastUser := users[len(users)-1]
		userID = lastUser["id"].(string)
		if userID == "" {
			t.Fatalf("expected user ID in response")
		}
	})

	// 3. Получение пользователя по ID
	t.Run("Get User by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users/"+userID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["id"] != userID {
			t.Fatalf("expected user ID %v, got %v", userID, response["id"])
		}

		// Проверка всех полей ответа
		for key, value := range createdUser {
			if key != "password" && response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 4. Обновление пользователя по ID
	t.Run("Update User by ID", func(t *testing.T) {
		updatedUser := map[string]interface{}{
			"display_name": fake.Person().Name(),
			"user_name":    fake.Gamer().Tag(),
			"role":         "player",
			"avatar_url":   "http://example.com/avatar_updated.png",
			"status":       "active",
			"password":     fake.Internet().Password(),
		}
		body, _ := json.Marshal(updatedUser)
		req, _ := http.NewRequest("PUT", "/api/v1/users/"+userID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["data"] != "User updated successfully" {
			t.Fatalf("expected success message, got %v", response["data"])
		}

		// Проверка всех полей ответа после обновления
		req, _ = http.NewRequest("GET", "/api/v1/users/"+userID, nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var updatedResponse map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &updatedResponse); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		for key, value := range updatedUser {
			if key != "password" && updatedResponse[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, updatedResponse[key])
			}
		}
	})

	// 5. Получение профиля пользователя по ID (его нет, поэтому ожидаем 404)
	t.Run("User Profile by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/users/"+userID+"/profile", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status code 404, got %v", rr.Code)
		}
	})

	// 6. Удаление пользователя по ID
	t.Run("Delete User by ID", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/users/"+userID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}
	})
}

func TestServiceCRUD(t *testing.T) {
	r, err := NewTestRouter()
	if err != nil {
		t.Fatalf("failed to initialize router: %v", err)
	}

	var serviceID string
	var createdService map[string]interface{}
	fake := faker.New()

	// 1. Создание сервиса
	t.Run("Create Service", func(t *testing.T) {
		createdService = map[string]interface{}{
			"name":        fake.Company().Name(),
			"author":      fake.Person().Name(),
			"logo_url":    fake.Internet().URL() + "image.png",
			"description": fake.Lorem().Sentence(10),
			"is_public":   fake.Bool(),
		}
		body, _ := json.Marshal(createdService)
		req, _ := http.NewRequest("POST", "/api/v1/services", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		serviceID = response["id"].(string)
		if serviceID == "" {
			t.Fatalf("expected service ID in response")
		}

		// Проверка всех полей ответа
		for key, value := range createdService {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 2. Получение всех сервисов и использование ID последнего
	t.Run("Get All Services", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/services", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var services []map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &services); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if len(services) == 0 {
			t.Fatalf("expected at least one service")
		}

		lastService := services[len(services)-1]
		serviceID = lastService["id"].(string)
		if serviceID == "" {
			t.Fatalf("expected service ID in response")
		}
	})

	// 3. Получение сервиса по ID
	t.Run("Get Service by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/services/"+serviceID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["id"] != serviceID {
			t.Fatalf("expected service ID %v, got %v", serviceID, response["id"])
		}

		// Проверка всех полей ответа
		for key, value := range createdService {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 4. Обновление сервиса по ID
	t.Run("Update Service by ID", func(t *testing.T) {
		updatedService := map[string]interface{}{
			"name":        fake.Company().Name(),
			"author":      fake.Person().Name(),
			"logo_url":    fake.Internet().URL() + "image.png",
			"description": fake.Lorem().Sentence(10),
			"is_public":   fake.Bool(),
		}
		body, _ := json.Marshal(updatedService)
		req, _ := http.NewRequest("PUT", "/api/v1/services/"+serviceID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["data"] != "Service updated successfully" {
			t.Fatalf("expected 'Service updated successfully', got %v", response["data"])
		}

		// Проверка всех полей ответа после обновления
		req, _ = http.NewRequest("GET", "/api/v1/services/"+serviceID, nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var updatedResponse map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &updatedResponse); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		for key, value := range updatedService {
			if updatedResponse[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, updatedResponse[key])
			}
		}
	})

	// 5. Удаление сервиса по ID
	t.Run("Delete Service by ID", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/services/"+serviceID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}
	})
}

func TestTeamCRUD(t *testing.T) {
	r, err := NewTestRouter()
	if err != nil {
		t.Fatalf("failed to initialize router: %v", err)
	}

	var teamID string
	var createdTeam map[string]interface{}
	fake := faker.New()

	// 1. Создание команды
	t.Run("Create Team", func(t *testing.T) {
		createdTeam = map[string]interface{}{
			"name":         fake.Gamer().Tag(),
			"description":  fake.Lorem().Sentence(10),
			"social_links": fake.Internet().URL(),
			"avatar_url":   fake.Internet().URL() + "image.png",
		}
		body, _ := json.Marshal(createdTeam)
		req, _ := http.NewRequest("POST", "/api/v1/teams", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		teamID = response["id"].(string)
		if teamID == "" {
			t.Fatalf("expected team ID in response")
		}

		// Проверка всех полей ответа
		for key, value := range createdTeam {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 2. Получение всех команд и использование ID последней
	t.Run("Get All Teams", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/teams", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var teams []map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &teams); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if len(teams) == 0 {
			t.Fatalf("expected at least one team")
		}

		lastTeam := teams[len(teams)-1]
		teamID = lastTeam["id"].(string)
		if teamID == "" {
			t.Fatalf("expected team ID in response")
		}
	})

	// 3. Получение команды по ID
	t.Run("Get Team by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/teams/"+teamID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["id"] != teamID {
			t.Fatalf("expected team ID %v, got %v", teamID, response["id"])
		}

		// Проверка всех полей ответа
		for key, value := range createdTeam {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 4. Обновление команды по ID
	t.Run("Update Team by ID", func(t *testing.T) {
		updatedTeam := map[string]interface{}{
			"name":         fake.Gamer().Tag(),
			"description":  fake.Lorem().Sentence(10),
			"social_links": fake.Internet().URL(),
			"avatar_url":   fake.Internet().URL() + "image.png",
		}
		body, _ := json.Marshal(updatedTeam)
		req, _ := http.NewRequest("PUT", "/api/v1/teams/"+teamID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["data"] != "Team updated successfully" {
			t.Fatalf("expected 'Team updated successfully, got %v", response["data"])
		}

		// Проверка всех полей ответа после обновления
		req, _ = http.NewRequest("GET", "/api/v1/teams/"+teamID, nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var updatedResponse map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &updatedResponse); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		for key, value := range updatedTeam {
			if updatedResponse[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, updatedResponse[key])
			}
		}
	})

	// 5. Удаление команды по ID
	t.Run("Delete Team by ID", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/teams/"+teamID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}
	})
}

func TestGameCRUD(t *testing.T) {
	r, err := NewTestRouter()
	if err != nil {
		t.Fatalf("failed to initialize router: %v", err)
	}

	var gameID string
	var createdGame map[string]interface{}
	fake := faker.New()

	// 1. Создание игры
	t.Run("Create Game", func(t *testing.T) {
		createdGame = map[string]interface{}{
			"start_time":  time.Date(2023, 7, 16, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			"end_time":    time.Date(2023, 7, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			"description": fake.Lorem().Sentence(10),
		}
		body, _ := json.Marshal(createdGame)
		req, _ := http.NewRequest("POST", "/api/v1/games", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		gameID = response["id"].(string)
		if gameID == "" {
			t.Fatalf("expected game ID in response")
		}

		// Проверка всех полей ответа
		for key, value := range createdGame {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 2. Получение всех игр и использование ID последней
	t.Run("Get All Games", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/games", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var games []map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &games); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if len(games) == 0 {
			t.Fatalf("expected at least one game")
		}

		lastGame := games[len(games)-1]
		gameID = lastGame["id"].(string)
		if gameID == "" {
			t.Fatalf("expected game ID in response")
		}
	})

	// 3. Получение игры по ID
	t.Run("Get Game by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/games/"+gameID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["id"] != gameID {
			t.Fatalf("expected game ID %v, got %v", gameID, response["id"])
		}

		// Проверка всех полей ответа
		for key, value := range createdGame {
			if response[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, response[key])
			}
		}
	})

	// 4. Обновление игры по ID
	t.Run("Update Game by ID", func(t *testing.T) {
		updatedGame := map[string]interface{}{
			"start_time":  time.Date(2023, 7, 16, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			"end_time":    time.Date(2023, 7, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			"description": fake.Lorem().Sentence(10),
		}
		body, _ := json.Marshal(updatedGame)
		req, _ := http.NewRequest("PUT", "/api/v1/games/"+gameID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["data"] != "Game updated successfully" {
			t.Fatalf("expected 'Game updated successfully', got %v", response["data"])
		}

		// Проверка всех полей ответа после обновления
		req, _ = http.NewRequest("GET", "/api/v1/games/"+gameID, nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var updatedResponse map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &updatedResponse); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		for key, value := range updatedGame {
			if updatedResponse[key] != value {
				t.Fatalf("expected %v for key %v, got %v", value, key, updatedResponse[key])
			}
		}
	})

	// 5. Удаление игры по ID
	t.Run("Delete Game by ID", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/games/"+gameID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}
	})
}

func TestResultsCRUD(t *testing.T) {
	r, err := NewTestRouter()
	if err != nil {
		t.Fatalf("failed to initialize router: %v", err)
	}

	var gameID, teamID string
	fake := faker.New()

	// 1. Создание игры и команды
	game := map[string]interface{}{
		"start_time":  time.Date(2023, 7, 16, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"end_time":    time.Date(2023, 7, 17, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"description": fake.Lorem().Sentence(10),
	}
	body, _ := json.Marshal(game)
	req, _ := http.NewRequest("POST", "/api/v1/games", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %v", rr.Code)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	gameID = response["id"].(string)
	if gameID == "" {
		t.Fatalf("expected game ID in response")
	}

	createdTeam := map[string]interface{}{
		"name":         fake.Gamer().Tag(),
		"description":  fake.Lorem().Sentence(10),
		"social_links": fake.Internet().URL(),
		"avatar_url":   fake.Internet().URL() + "image.png",
	}
	body, _ = json.Marshal(createdTeam)
	req, _ = http.NewRequest("POST", "/api/v1/teams", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %v", rr.Code)
	}

	// var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}

	teamID = response["id"].(string)
	if teamID == "" {
		t.Fatalf("expected team ID in response")
	}

	var resultID string

	// 2. Создание результата игры
	t.Run("Create Result", func(t *testing.T) {
		result := map[string]interface{}{
			"game_id": gameID,
			"rank":    fake.IntBetween(1, 10),
			"score":   fake.Float64(0, 1, 100),
			"team_id": teamID,
		}
		body, _ := json.Marshal(result)
		req, _ := http.NewRequest("POST", "/api/v1/games/"+gameID+"/results", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		resultID = response["id"].(string)
		if resultID == "" {
			t.Fatalf("expected result ID in response")
		}
	})

	// 3. Получение результата игры по ID
	t.Run("Get Result by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/games/"+gameID+"/results", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if len(response) == 0 {
			t.Fatalf("expected at least one result")
		}
	})

	// 4. Обновление результата игры по ID
	t.Run("Update Result by ID", func(t *testing.T) {
		t.Skip()
		updatedResult := map[string]interface{}{
			"rank":  fake.IntBetween(1, 10),
			"score": fake.Float64(0, 1, 100),
		}
		body, _ := json.Marshal(updatedResult)
		req, _ := http.NewRequest("PUT", "/api/v1/games/"+gameID+"/results", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if response["data"] != "Result updated successfully" {
			t.Fatalf("expected 'Result updated successfully', got %v", response["data"])
		}
	})

	// 5. Получение игровой таблицы результатов
	t.Run("Get Game Scoreboard", func(t *testing.T) {
		t.Skip()
		req, _ := http.NewRequest("GET", "/api/v1/games/"+gameID+"/scoreboard", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code 200, got %v", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}
	})
}
