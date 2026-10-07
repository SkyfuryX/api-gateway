package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/SkyfuryX/api-gateway/internal/admin"
	"github.com/SkyfuryX/api-gateway/internal/tenants"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, relying on system environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	repo := tenants.NewRepository(nil, dbPool)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("========================================")
	fmt.Println("     API Gateway Admin Console          ")
	fmt.Println("========================================")

	for {
		fmt.Println("\nSelect an option:")
		fmt.Println("1) Create New Tenant & API Key")
		fmt.Println("2) Update Tenant Tier & Rate")
		fmt.Println("3) Update Tenant Key Status")
		fmt.Println("4) Issue New API Key")
		fmt.Println("5) Exit")
		fmt.Print("> ")

		choice, _ := reader.ReadString('\n')
		choice = strings.ToLower(strings.TrimSpace(choice))

		switch choice {
		case "1":
			admin.PromptCreateTenant(reader, repo)
		case "2":
			admin.PromptUpdateTenantTierRate(reader, repo)
		case "3":
			admin.PromptUpdateTenantStatus(reader, repo)
		case "4":
			admin.PromptUpdateAPIKey(reader, repo)
		case "5", "exit", "quit":
			fmt.Println("Exiting admin console. Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("Invalid Input")
		}

	}

}
