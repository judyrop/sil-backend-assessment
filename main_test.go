package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/judyrop/sil-backend/models"
)

// Create DB connection for tests
func getTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	db.AutoMigrate(&models.Customer{}, &models.Category{}, &models.Product{}, &models.Order{})
	return db
}

// Helper: run a test inside a transaction and roll it back
func withTestTransaction(t *testing.T, testFunc func(tx *gorm.DB)) {
	db := getTestDB()

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}

	defer tx.Rollback() 

	testFunc(tx) 
}

// ----------------------- TESTS ----------------------- //

func TestCreateCategory(t *testing.T) {
	withTestTransaction(t, func(db *gorm.DB) {
		router := SetupRouter(db)

		category := map[string]interface{}{
			"name": "Bakery",
		}
		body, _ := json.Marshal(category)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/categories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code) 
	})
}

func TestCreateCustomer(t *testing.T) {
	withTestTransaction(t, func(db *gorm.DB) {
		router := SetupRouter(db)

		customer := map[string]interface{}{
			"name":  "June Jun",
			"email": "junejun@gmail.com",
			"phone": "0712345678",
		}
		body, _ := json.Marshal(customer)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp models.Customer
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "June Jun", resp.Name)
		assert.Equal(t, "junejun@gmail.com", resp.Email)
		assert.Equal(t, "0712345678", resp.Phone)
	})
}

func TestListCategories(t *testing.T) {
	withTestTransaction(t, func(db *gorm.DB) {
		router := SetupRouter(db)

		// Create a category first
		db.Create(&models.Category{Name: "Bakery"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/categories", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []models.Category
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp)
		assert.Equal(t, "Bakery", resp[0].Name)
	})
}

func TestCreateProduct(t *testing.T) {
	withTestTransaction(t, func(db *gorm.DB) {
		router := SetupRouter(db)

		// Create a category for the product
		category := models.Category{Name: "Bakery"}
		db.Create(&category)
		

		product := map[string]interface{}{
			"name":        "Bread",
			"price":       3.5,
			"category_id": category.ID,
		}
		body, _ := json.Marshal(product)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp models.Product
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Bread", resp.Name)
		assert.Equal(t, category.ID, resp.CategoryID)
	})
}


