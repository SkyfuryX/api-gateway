package admin

import (
	"bufio"
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"github.com/SkyfuryX/api-gateway/internal/db"
	"github.com/SkyfuryX/api-gateway/internal/tenants"
)

func PromptCreateTenant(reader *bufio.Reader, repo *tenants.Repository) {
	fmt.Println("\n--- Create New Tenant ---")
	fmt.Println("Use 'back' to return to the previous menu.")

	for {
		name, exit := promptName(reader)
		if exit {
			return
		}

		email, exit := promptEmail(reader)
		if exit {
			return
		}

		tenantTier, exit := promptTier(reader)
		if exit {
			return
		}

		rateLimit, exit := promptRateLimit(reader)
		if exit {
			return
		}

		// Confirm details
		fmt.Println("\nTenant Details:")
		fmt.Println("----------------------------------------")
		fmt.Printf("Name:       %s\n", name)
		fmt.Printf("Email:      %s\n", email)
		fmt.Printf("Tier:       %s\n", tenantTier)
		fmt.Printf("Rate Limit: %d req/min\n", rateLimit)

		confirm, exit := promptConfirm(reader)
		if exit {
			return
		}
		if !confirm {
			continue
		}

		// Create in Database
		tenant, err := repo.CreateTenant(name, email, tenantTier, rateLimit)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Output Results
		fmt.Println("\nTenant Successfully Created!")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:         %s\n", tenant.ID)
		fmt.Printf("Name:       %s\n", tenant.Name)
		fmt.Printf("Email:      %s\n", tenant.Email)
		fmt.Printf("Tier:       %s\n", tenant.Tier)
		fmt.Printf("Rate Limit: %d req/min\n", tenant.RateLimitReqPerMin)
		fmt.Printf("API Key:    %s\n", tenant.ApiKey)
		fmt.Println("----------------------------------------")
		fmt.Println("Save this API key. It will not be shown again.")

		return
	}
}

func PromptUpdateTenantStatus(reader *bufio.Reader, repo *tenants.Repository) {
	for {
		fmt.Println("\n--- Activate or Suspend a Tenant ---")
		fmt.Println("Use 'back' to return to the previous menu.")

		tenant, exit := promptTenantLookup(reader, repo)
		if exit {
			return
		}

		status, exit := promptStatus(reader)
		if exit {
			return
		}

		// Confirm details
		fmt.Println("\nTenant Details:")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:         %s\n", tenant.ID)
		fmt.Printf("Name:       %s\n", tenant.Name)
		fmt.Printf("Email:      %s\n", tenant.Email)
		fmt.Printf("API Key:    %s\n", tenant.ApiKey)
		fmt.Printf("Tier:       %s\n", tenant.Tier)
		fmt.Println("----------------------------------------")
		fmt.Printf("This tenant will be set to: '%s'\n", string(status))
		confirm, exit := promptConfirm(reader)
		if exit {
			return
		}
		if !confirm {
			continue
		}

		result, err := repo.UpdateTenantStatus(tenant.ID, status)
		if err != nil {
			println(err)
			return
		}

		// Output Results
		fmt.Println("\nTenant Successfully Updated!")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:         %s\n", result.ID)
		fmt.Printf("Status      %s\n", result.Status)
		fmt.Println("----------------------------------------")

		// clears cached key in Redis on status change
		if result.Status != tenant.Status {
			if err = repo.InvalidateCache(context.Background(), result.ApiKey); err != nil {
				fmt.Println(err)
			}
		}

		return
	}
}

func PromptUpdateTenantTierRate(reader *bufio.Reader, repo *tenants.Repository) {
	for {
		fmt.Println("\n--- Update a Tenant's Rate Limit ---")
		fmt.Println("Use 'back' to return to the previous menu.")

		tenant, exit := promptTenantLookup(reader, repo)
		if exit {
			return
		}

		rate, exit := promptRateLimit(reader)
		if exit {
			return
		}

		tier, exit := promptTier(reader)
		if exit {
			return
		}

		// Confirm details
		fmt.Println("\nTenant Details:")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:         %s\n", tenant.ID)
		fmt.Printf("Name:       %s\n", tenant.Name)
		fmt.Printf("Email:      %s\n", tenant.Email)
		fmt.Printf("API Key:    %s\n", tenant.ApiKey)
		fmt.Println("----------------------------------------")
		fmt.Printf("This tenant updated to the '%s' tier with '%d' requests/min.\n", tier, rate)
		confirm, exit := promptConfirm(reader)
		if exit {
			return
		}
		if !confirm {
			continue
		}

		result, err := repo.UpdateTierAndRate(tenant.ID, tier, rate)
		if err != nil {
			println(err)
			return
		}

		// Output Results
		fmt.Println("\nTenant Successfully Updated!")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:        %s\n", result.ID)
		fmt.Printf("Tier:      %s\n", result.Tier)
		fmt.Printf("Rate:      %d\n", result.RateLimitReqPerMin)
		fmt.Println("----------------------------------------")

		return
	}

}

func PromptUpdateAPIKey(reader *bufio.Reader, repo *tenants.Repository) {
	for {
		fmt.Println("\n--- Update a Tenant's API Key ---")
		fmt.Println("Use 'back' to return to the previous menu.")

		tenant, exit := promptTenantLookup(reader, repo)
		if exit {
			return
		}

		// Confirm details
		fmt.Println("\nTenant Details:")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:         %s\n", tenant.ID)
		fmt.Printf("Name:       %s\n", tenant.Name)
		fmt.Printf("Email:      %s\n", tenant.Email)
		fmt.Println("----------------------------------------")
		fmt.Println("This tenant will be issued a new API Key. The previous API Key will be invalid.")
		confirm, exit := promptConfirm(reader)
		if exit {
			return
		}
		if !confirm {
			continue
		}

		result, err := repo.UpdateTenantAPIKey(tenant.ID)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("\nTenant Successfully Updated!")
		fmt.Println("----------------------------------------")
		fmt.Printf("ID:         %s\n", result.ID)
		fmt.Printf("Name:       %s\n", result.Name)
		fmt.Printf("Email:      %s\n", result.Email)
		fmt.Printf("API Key:    %s\n", result.ApiKey)
		fmt.Printf("Updated:    %v\n", result.UpdatedAt)
		fmt.Println("----------------------------------------")
		fmt.Println("Save this API key. It will not be shown again.")

		return
	}
}

func promptTenantLookup(reader *bufio.Reader, repo *tenants.Repository) (db.Tenant, bool) {
	for {
		fmt.Println("1) API Key")
		fmt.Println("2) Email")
		fmt.Print("Choose Tenant Lookup Method: ")

		input, exit := readInput(reader)
		if exit {
			return db.Tenant{}, true
		}
		input = strings.ToLower(input)

		switch input {
		case "1", "api key":
			apiKey, exit := promptAPIKey(reader)
			if exit {
				return db.Tenant{}, true
			}

			tenant, err := repo.Queries.GetTenantByAPIKey(context.Background(), apiKey)
			if err != nil {
				fmt.Println(err)
				continue
			}
			return tenant, false
		case "2", "email":
			email, exit := promptEmail(reader)
			if exit {
				return db.Tenant{}, true
			}

			tenant, err := repo.Queries.GetTenantByEmail(context.Background(), email)
			if err != nil {
				fmt.Println(err)
				continue
			}
			return tenant, false
		default:
			fmt.Println("Invalid Input")
			continue
		}
	}
}

func promptName(reader *bufio.Reader) (string, bool) {
	// Read Name, continue until non-empty string provided
	for {
		fmt.Print("\nNew Tenant Name: ")
		name, exit := readInput(reader)
		if exit {
			return "", true
		}
		if name == "" {
			fmt.Println("Error: Name cannot be empty.")
			continue
		}
		return name, false
	}
}

func promptRateLimit(reader *bufio.Reader) (rateLimit int, exit bool) {
	fmt.Print("Enter Rate Limit (requests per minute, default: 60): ")
	input, exit := readInput(reader)
	if exit {
		return 0, true
	}

	rateLimit = 60
	if input != "" {
		parsed, err := strconv.Atoi(input)
		if err != nil || parsed <= 0 {
			fmt.Println("Invalid rate limit number, using default of 60.")
		} else {
			rateLimit = parsed
		}
	}
	return rateLimit, false
}

func promptAPIKey(reader *bufio.Reader) (apiKey string, exit bool) {
	for {
		fmt.Print("Tenant API Key: ")
		input, exit := readInput(reader)
		if exit {
			return "", true
		}

		if input == "" {
			fmt.Println("Error: Invalid API Key.")
			continue
		}

		return input, false
	}
}

func promptEmail(reader *bufio.Reader) (email string, exit bool) {
	for {
		fmt.Print("Tenant Email: ")
		input, exit := readInput(reader)
		if exit {
			return "", true
		}

		parsedEmail, err := mail.ParseAddress(input)
		if input == "" || err != nil {
			fmt.Println("Error: Invalid Email.")
			continue
		}
		email := parsedEmail.Address
		return email, false
	}
}

// Confirm user input before proceeding
func promptConfirm(reader *bufio.Reader) (confirm bool, exit bool) {
	for {
		fmt.Print("Confirm? [y/n]: ")
		input, exit := readInput(reader)
		if exit {
			return false, true
		}
		switch strings.ToLower(input) {
		case "n", "no":
			return false, false
		case "y", "yes":
			return true, false
		}
	}
}

// Read/Parse Tier into type-safe input
func promptTier(reader *bufio.Reader) (tier tenants.TenantTier, exit bool) {
	fmt.Println("1) Free")
	fmt.Println("2) Pro")
	fmt.Println("3) Enterprise")
	fmt.Print("Tenant Tier (default: Free): ")
	input, exit := readInput(reader)
	if exit {
		return "", true
	}
	switch strings.ToLower(input) {
	case "1", "free":
		return tenants.TierFree, false
	case "2", "pro":
		return tenants.TierPro, false
	case "3", "enterprise":
		return tenants.TierEnterprise, false
	default:
		return tenants.TierFree, false
	}
}

// Read/Parse Status into type-safe input
func promptStatus(reader *bufio.Reader) (status tenants.TenantStatus, exit bool) {
	for {
		fmt.Println("1) Active")
		fmt.Println("2) Suspended")
		fmt.Print("User Status: ")
		input, exit := readInput(reader)
		if exit {
			return "", true
		}
		switch strings.ToLower(input) {
		case "1", "active":
			return tenants.StatusActive, false
		case "2", "suspended":
			return tenants.StatusSuspended, false
		default:
			fmt.Println("Invalid Status (ex: use '1'/'2' or 'active'/'suspended').")
			continue
		}
	}
}

// readInput reads raw text, trims whitespace
func readInput(reader *bufio.Reader) (input string, isBack bool) {
	raw, _ := reader.ReadString('\n')
	input = strings.TrimSpace(raw)
	if cliBackCheck(input) {
		return "", true
	}
	return input, false
}
