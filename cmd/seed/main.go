// cmd/seed/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Config validation failed:", err)
	}

	// Initialize database
	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	ctx := context.Background()

	log.Println("🌱 Starting database seeding...")

	// Check if we should clear existing data
	clearData := os.Getenv("CLEAR_DATA") == "true"
	if clearData {
		log.Println("🗑️  Clearing existing data...")
		if err := clearExistingData(ctx, db); err != nil {
			log.Fatal("Failed to clear existing data:", err)
		}
	}

	// Seed products
	if err := seedProducts(ctx, db); err != nil {
		log.Fatal("Failed to seed products:", err)
	}

	// Seed customers
	if err := seedCustomers(ctx, db); err != nil {
		log.Fatal("Failed to seed customers:", err)
	}

	// Seed settings
	if err := seedSettings(ctx, db); err != nil {
		log.Fatal("Failed to seed settings:", err)
	}

	log.Println("✅ Database seeding completed successfully!")
}

func clearExistingData(ctx context.Context, db *database.DB) error {
	log.Println("Clearing cart_items...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM cart_items"); err != nil {
		return err
	}

	log.Println("Clearing carts...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM carts"); err != nil {
		return err
	}

	log.Println("Clearing order_items...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM order_items"); err != nil {
		return err
	}

	log.Println("Clearing orders...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM orders"); err != nil {
		return err
	}

	log.Println("Clearing customers...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM customers"); err != nil {
		return err
	}

	log.Println("Clearing products...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM products"); err != nil {
		return err
	}

	log.Println("Clearing events...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM events"); err != nil {
		return err
	}

	log.Println("Clearing audit_logs...")
	if _, err := db.Conn().Exec(ctx, "DELETE FROM audit_logs"); err != nil {
		return err
	}

	return nil
}

func seedProducts(ctx context.Context, db *database.DB) error {
	log.Println("🔸 Seeding products...")

	products := []struct {
		name        string
		description string
		price       int32 // cents
		stock       int32
		active      bool
	}{
		{
			name:        "Ethiopian Yirgacheffe",
			description: "Bright, floral notes with citrus finish. Grown at high altitude in the Sidama region.",
			price:       1800, // $18.00
			stock:       25,
			active:      true,
		},
		{
			name:        "Colombian Supremo",
			description: "Rich, full-bodied with chocolate undertones. Single-origin from the Huila region.",
			price:       1600, // $16.00
			stock:       32,
			active:      true,
		},
		{
			name:        "Guatemala Antigua",
			description: "Medium body with spicy and smoky flavors. Volcanic soil creates unique complexity.",
			price:       1750, // $17.50
			stock:       18,
			active:      true,
		},
		{
			name:        "Brazilian Santos",
			description: "Smooth, nutty flavor with low acidity. Perfect for espresso blends.",
			price:       1400, // $14.00
			stock:       45,
			active:      true,
		},
		{
			name:        "Jamaica Blue Mountain",
			description: "Legendary coffee with mild flavor and no bitterness. Extremely limited production.",
			price:       4500, // $45.00
			stock:       8,
			active:      true,
		},
		{
			name:        "Costa Rican Tarrazú",
			description: "Full-bodied with bright acidity and wine-like flavor. High-altitude grown.",
			price:       1950, // $19.50
			stock:       22,
			active:      true,
		},
		{
			name:        "Kenya AA",
			description: "Bold, wine-like acidity with black currant notes. Distinctively African character.",
			price:       2100, // $21.00
			stock:       15,
			active:      true,
		},
		{
			name:        "Hawaiian Kona",
			description: "Smooth, rich flavor with low acidity. Grown on volcanic slopes of the Big Island.",
			price:       3500, // $35.00
			stock:       12,
			active:      true,
		},
		{
			name:        "House Blend",
			description: "Our signature blend combining Ethiopian and Colombian beans for perfect balance.",
			price:       1500, // $15.00
			stock:       50,
			active:      true,
		},
		{
			name:        "Dark Roast Espresso",
			description: "Bold, intense flavor perfect for espresso drinks. Low acidity with smoky notes.",
			price:       1650, // $16.50
			stock:       35,
			active:      true,
		},
		{
			name:        "Decaf Colombian",
			description: "Swiss water processed Colombian coffee. All the flavor, none of the caffeine.",
			price:       1700, // $17.00
			stock:       20,
			active:      true,
		},
		{
			name:        "Seasonal Blend - Winter",
			description: "Limited edition winter blend with cinnamon and nutmeg notes.",
			price:       1800, // $18.00
			stock:       0,    // Out of stock
			active:      false,
		},
	}

	for _, p := range products {
		description := pgtype.Text{
			String: p.description,
			Valid:  true,
		}

		created, err := db.Queries.CreateProduct(ctx, database.CreateProductParams{
			Name:        p.name,
			Description: description,
			Price:       p.price,
			Stock:       p.stock,
			Active:      p.active,
		})
		if err != nil {
			return err
		}

		log.Printf("  ✓ Created product: %s (ID: %d)", created.Name, created.ID)
	}

	return nil
}

func seedCustomers(ctx context.Context, db *database.DB) error {
	log.Println("👤 Seeding customers...")

	customers := []struct {
		email        string
		firstName    string
		lastName     string
		passwordHash string
	}{
		{
			email:        "john.doe@example.com",
			firstName:    "John",
			lastName:     "Doe",
			passwordHash: "$2a$10$placeholder.hash.for.password123", // In real app, hash "password123"
		},
		{
			email:        "jane.smith@example.com",
			firstName:    "Jane",
			lastName:     "Smith",
			passwordHash: "$2a$10$placeholder.hash.for.password456",
		},
		{
			email:        "coffee.lover@example.com",
			firstName:    "Coffee",
			lastName:     "Lover",
			passwordHash: "$2a$10$placeholder.hash.for.password789",
		},
		{
			email:        "admin@coffeeshop.com",
			firstName:    "Admin",
			lastName:     "User",
			passwordHash: "$2a$10$placeholder.hash.for.admin123",
		},
	}

	for _, c := range customers {
		firstName := pgtype.Text{String: c.firstName, Valid: true}
		lastName := pgtype.Text{String: c.lastName, Valid: true}

		created, err := db.Queries.CreateCustomer(ctx, database.CreateCustomerParams{
			Email:        c.email,
			FirstName:    firstName,
			LastName:     lastName,
			PasswordHash: c.passwordHash,
		})
		if err != nil {
			return err
		}

		log.Printf("  ✓ Created customer: %s %s <%s> (ID: %d)",
			created.FirstName.String, created.LastName.String, created.Email, created.ID)
	}

	return nil
}

func seedSettings(ctx context.Context, db *database.DB) error {
	log.Println("⚙️  Seeding settings...")

	// Check if settings already exist (they're created in migration)
	var count int
	err := db.Conn().QueryRow(ctx, "SELECT COUNT(*) FROM settings").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("  ℹ️  Settings already exist, skipping...")
		return nil
	}

	settings := []struct {
		key         string
		value       string
		description string
		category    string
	}{
		{
			key:         "tax_rate",
			value:       "0.08",
			description: "Default tax rate for orders (8%)",
			category:    "pricing",
		},
		{
			key:         "free_shipping_threshold",
			value:       "5000",
			description: "Free shipping threshold in cents ($50)",
			category:    "shipping",
		},
		{
			key:         "currency",
			value:       "\"USD\"",
			description: "Default currency code",
			category:    "general",
		},
		{
			key:         "site_name",
			value:       "\"Artisan Coffee Roasters\"",
			description: "Site name for emails and branding",
			category:    "general",
		},
		{
			key:         "support_email",
			value:       "\"support@artisancoffee.com\"",
			description: "Customer support email",
			category:    "general",
		},
		{
			key:         "low_stock_threshold",
			value:       "10",
			description: "Alert when product stock falls below this number",
			category:    "inventory",
		},
		{
			key:         "max_cart_items",
			value:       "50",
			description: "Maximum number of items allowed in cart",
			category:    "general",
		},
	}

	for _, s := range settings {
		description := pgtype.Text{String: s.description, Valid: true}

		_, err := db.Conn().Exec(ctx,
			"INSERT INTO settings (key, value, description, category) VALUES ($1, $2, $3, $4)",
			s.key, s.value, description, s.category)
		if err != nil {
			return err
		}

		log.Printf("  ✓ Created setting: %s = %s", s.key, s.value)
	}

	return nil
}
