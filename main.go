package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/judyrop/sil-backend/models"
)

var (
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
)

const (
	apiKey   = "atsk_d33f6bbab9be3346bd343b3ace98d8c85c7bb5e3b9ee714794953b67dea7863671e05ef7"
	username = "sandbox"
)

func main() {
dsn := "host=postgres user=postgres password=postgres dbname=sil port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Category{}, &models.Product{}, &models.Order{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	initOIDC()
	r := SetupRouter(db)
	r.Run(":8080")
}

func initOIDC() {
	var err error
	provider, err = oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		log.Fatal(err)
	}
	verifier = provider.Verifier(&oidc.Config{ClientID: "896138674473-nv0aunk26qg7v4hj6pgkna26chqnikg7.apps.googleusercontent.com"})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		token := strings.TrimPrefix(authHeader, prefix)
		if _, err := verifier.Verify(context.Background(), token); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		c.Next()
	}
}

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Create product
	r.POST("/products", func(c *gin.Context) {
		var product models.Product
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if db != nil {
			if err := db.Create(&product).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusCreated, product)
	})

	// Create category
	r.POST("/categories", func(c *gin.Context) {
		var category models.Category
		if err := c.ShouldBindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if db != nil {
			if err := db.Create(&category).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusCreated, category)
	})

	// List categories
	r.GET("/categories", func(c *gin.Context) {
		var categories []models.Category
		if db != nil {
			if err := db.Find(&categories).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, categories)
	})

	// Average price for category
	r.GET("/categories/:id/average-price", func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusOK, gin.H{"average_price": 0})
			return
		}
		id := c.Param("id")
		var category models.Category
		if err := db.First(&category, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}
		allCatIDs, err := getDescendantCategoryIDs(db, category.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var products []models.Product
		if err := db.Where("category_id IN ?", allCatIDs).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var sum float64
		for _, p := range products {
			sum += p.Price
		}
		avg := 0.0
		if len(products) > 0 {
			avg = sum / float64(len(products))
		}
		c.JSON(http.StatusOK, gin.H{
			"category_id":   id,
			"category_name": category.Name,
			"average_price": avg,
		})
	})

	// Create an order
	r.POST("/orders", AuthMiddleware(), func(c *gin.Context) {
		var req struct {
			CustomerID uint   `json:"customer_id"`
			ProductIDs []uint `json:"product_ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if db == nil {
			c.JSON(http.StatusCreated, gin.H{"mock": "order"})
			return
		}

		var customer models.Customer
		if err := db.First(&customer, req.CustomerID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}
		var products []models.Product
		if err := db.Where("id IN ?", req.ProductIDs).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var total float64
		for _, p := range products {
			total += p.Price
		}
		order := models.Order{CustomerID: customer.ID, Products: products, Total: total}
		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// notifications
		sendSMS(customer.Phone, fmt.Sprintf("Hi %s, your order #%d has been placed!", customer.Name, order.ID))
		sendOrderEmail(order, customer, products)

		c.JSON(http.StatusCreated, order)
	})

	// Create customer
	r.POST("/customers", func(c *gin.Context) {
		var customer models.Customer
		if err := c.ShouldBindJSON(&customer); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Create(&customer).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, customer)
	})

	return r
}

func sendSMS(phone, message string) error {
	data := url.Values{}
	data.Set("username", username)
	data.Set("to", phone)
	data.Set("message", message)
	req, err := http.NewRequest("POST", "https://api.sandbox.africastalking.com/version1/messaging", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("apiKey", apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("SMS Response:", string(body))
	return nil
}

func sendOrderEmail(order models.Order, customer models.Customer, products []models.Product) error {
	from := "cocoa@gmail.com"
	pass := "cocoa"
	to := "Hazel@gmail.com"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	body := fmt.Sprintf("Order #%d placed by %s (%s)\nProducts: %+v\nTotal: %.2f",
		order.ID, customer.Name, customer.Email, products, order.Total)
	msg := "From: " + from + "\n" + "To: " + to + "\n" + "Subject: New Order Placed\n\n" + body
	auth := smtp.PlainAuth("", from, pass, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
}

func getDescendantCategoryIDs(db *gorm.DB, categoryID uint) ([]uint, error) {
	var ids []uint
	ids = append(ids, categoryID)
	var children []models.Category
	if err := db.Where("parent_id = ?", categoryID).Find(&children).Error; err != nil {
		return nil, err
	}
	for _, child := range children {
		childIDs, err := getDescendantCategoryIDs(db, child.ID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, childIDs...)
	}
	return ids, nil
}
