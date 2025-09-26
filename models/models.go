package models

import "gorm.io/gorm"

type Customer struct {
	gorm.Model
	Name   string
	Email  string
	Phone  string
	Orders []Order
}

type Category struct {
	gorm.Model
	Name     string
	ParentID *uint
	Products []Product
	Children []Category `gorm:"foreignKey:ParentID"`
}

type Product struct {
    gorm.Model
    Name       string  `json:"name"`
    Price      float64 `json:"price"`
    CategoryID uint    `json:"category_id"`
}


type Order struct {
	gorm.Model
	CustomerID uint
	Customer   Customer
	Products   []Product `gorm:"many2many:order_products;"`
	Total      float64
}
