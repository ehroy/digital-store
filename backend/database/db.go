package database

import (
	"digistore/config"
	"digistore/models"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.App.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := DB.AutoMigrate(
		&models.Product{},
		&models.ProductStock{},
		&models.Order{},
		&models.PaymentConfig{},
		&models.ScriptLog{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	seed()
	log.Println("✅ Database ready:", config.App.DBPath)
}
func seed() {
	var count int64

	DB.Model(&models.PaymentConfig{}).Count(&count)
	if count == 0 {
		DB.Create(&models.PaymentConfig{
			BankName: "BCA", BankNo: "8800123456", BankAcc: "PT Toko Digital Nusantara",
			Dana: "081234567890", Gopay: "081234567890", Ovo: "081234567890", QRIS: true,
		})
	}

	DB.Model(&models.Product{}).Count(&count)
	if count == 0 {

		p1 := models.Product{
			Name: "Account Tomoro Premium",
			Description: "Akun Tomoro siap pakai dengan benefit voucher eksklusif seperti Buy 1 Get 1 (B1G1) dan diskon hingga 50%. Cocok untuk kamu yang ingin menikmati kopi favorit dengan harga lebih hemat.\n\nGaransi 1x24 jam jika akun tidak memiliki voucher.",
			Price: 1500,
			Category: "Account",
			Type: "stock",
			Active: true,
			Icon: "☕",
		}

		p2 := models.Product{
			Name: "Account Kopi Kenangan Premium",
			Description: "Akun Kopi Kenangan siap digunakan dengan berbagai voucher menarik seperti B1G1 dan diskon 50%. Nikmati berbagai menu favorit dengan harga lebih murah.\n\nGaransi 1x24 jam jika akun tidak terdapat voucher.",
			Price: 1500,
			Category: "Account",
			Type: "stock",
			Active: true,
			Icon: "🥤",
		}

		// insert products
		for _, p := range []*models.Product{&p1, &p2} {
			DB.Create(p)
		}

		// Seed stock (akun contoh)
		stockData := map[uint][]string{
			p1.ID: {
				"email:tomoro1@mail.com|pass:123456",
				"email:tomoro2@mail.com|pass:123456",
				"email:tomoro3@mail.com|pass:123456",
			},
			p2.ID: {
				"email:kenangan1@mail.com|pass:123456",
				"email:kenangan2@mail.com|pass:123456",
				"email:kenangan3@mail.com|pass:123456",
			},
		}

		for pid, items := range stockData {
			for _, data := range items {
				DB.Create(&models.ProductStock{
					ProductID: pid,
					Data:      data,
				})
			}
		}

		log.Println("✅ Seed 2 produk akun premium berhasil")
	}
}