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
		&models.StockProvider{},
		&models.PullLog{},
		&models.ExternalProvider{},
		&models.ProviderProduct{},
		&models.ProviderOrder{},
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

	var cc models.ContactConfig
	DB.FirstOrCreate(&cc, models.ContactConfig{ID: 1})
	if cc.BusinessName == "" { cc.BusinessName = "DigiStore"; DB.Save(&cc) }

	DB.Model(&models.Product{}).Count(&count)
	if count == 0 {
		p1 := models.Product{Name: "Template Website Bisnis Pro", Description: "Template modern 12 halaman, SEO-ready, dokumentasi lengkap.", Price: 149000, Category: "Template", Type: "stock", Active: true, Icon: "🌐"}
		p2 := models.Product{Name: "Ebook SEO Mastery 2026", Description: "250 halaman panduan SEO, studi kasus nyata.", Price: 79000, Category: "Ebook", Type: "stock", Active: true, Icon: "📚"}
		p3 := models.Product{Name: "Plugin WordPress SEO Turbo", Description: "Plugin SEO premium: auto sitemap, schema markup.", Price: 129000, Category: "Plugin", Type: "stock", Active: true, Icon: "⚡"}
		p4 := models.Product{
			Name: "Layanan Desain Logo Premium", Description: "Logo profesional, revisi unlimited, format AI/PNG/SVG.",
			Price: 299000, Category: "Jasa", Type: "script", Active: true, Icon: "🎨",
			Script: `[{"provider":"email","to":"admin@example.com","subject":"🎨 Order Logo: {{invoice_no}}","body":"Pembeli: {{buyer_name}} ({{buyer_email}})\nTotal: {{total}}\nSegera hubungi klien."},{"provider":"log","message":"Tiket desain dibuat untuk {{buyer_name}} — invoice {{invoice_no}}"}]`,
		}
		for _, p := range []*models.Product{&p1, &p2, &p3, &p4} {
			DB.Create(p)
		}
		// Seed stock items
		stockData := map[uint][]string{
			p1.ID: {
				"https://drive.google.com/file/d/TEMPLATE_CONTOH_1/view",
				"https://drive.google.com/file/d/TEMPLATE_CONTOH_2/view",
				"https://drive.google.com/file/d/TEMPLATE_CONTOH_3/view",
			},
			p2.ID: {
				"https://drive.google.com/file/d/EBOOK_CONTOH_1/view",
				"https://drive.google.com/file/d/EBOOK_CONTOH_2/view",
				"https://drive.google.com/file/d/EBOOK_CONTOH_3/view",
				"https://drive.google.com/file/d/EBOOK_CONTOH_4/view",
			},
			p3.ID: {
				"LIC-WPSEO-AAAA-1111-XXXX",
				"LIC-WPSEO-BBBB-2222-XXXX",
				"LIC-WPSEO-CCCC-3333-XXXX",
			},
		}
		for pid, items := range stockData {
			for _, data := range items {
				DB.Create(&models.ProductStock{ProductID: pid, Data: data})
			}
		}
		log.Println("✅ Seed data inserted")
	}
}
