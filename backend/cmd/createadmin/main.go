// createadmin creates the first administrator (a SUPER_ADMIN) on a real server, or resets that
// account's password. Use it instead of loading the demo data.
//
//	docker compose run --rm --entrypoint /bin/createadmin \
//	  -e ADMIN_EMAIL=you@yourshop.com -e ADMIN_PASSWORD='a long password phrase' api
//
// Optional: ADMIN_FIRST_NAME, ADMIN_LAST_NAME. If the email already exists the command refuses,
// unless ADMIN_RESET=1 is set (then it sets the new password and re-activates the account).
// The password is never printed or logged.
package main

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	authInfra "github.com/ecommerce/backend/internal/auth/infrastructure"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	userInfra "github.com/ecommerce/backend/internal/users/infrastructure"
	"github.com/ecommerce/backend/pkg/config"
	"github.com/ecommerce/backend/pkg/mongodb"
)

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "createadmin: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := os.Getenv("ADMIN_PASSWORD")
	first := strings.TrimSpace(os.Getenv("ADMIN_FIRST_NAME"))
	last := strings.TrimSpace(os.Getenv("ADMIN_LAST_NAME"))
	reset := os.Getenv("ADMIN_RESET") == "1"

	if first == "" {
		first = "Admin"
	}
	if last == "" {
		last = "Owner"
	}
	if _, err := mail.ParseAddress(email); err != nil {
		fail("set ADMIN_EMAIL to a valid email address")
	}
	if len(password) < 12 {
		fail("ADMIN_PASSWORD must be at least 12 characters (a long phrase is best)")
	}
	if strings.EqualFold(password, "Password123!") {
		fail("that is the well-known demo password; choose another")
	}

	cfg, err := config.Load()
	if err != nil {
		fail("could not read the configuration: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongodb.New(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.MaxPoolSize, cfg.MongoDB.MinPoolSize, cfg.MongoDB.ConnectTimeout)
	if err != nil {
		fail("could not connect to the database: %v", err)
	}
	defer client.Close(ctx)
	db := client.Database()

	if err := userInfra.EnsureIndexes(ctx, db); err != nil {
		fail("could not prepare the users collection: %v", err)
	}
	users := userInfra.NewUserRepository(db)
	hash, err := authInfra.NewPasswordHasher(authInfra.DefaultCost).Hash(password)
	if err != nil {
		fail("could not secure the password: %v", err)
	}

	existing, err := users.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		if !reset {
			fail("%s already exists. To set a new password for it, run again with ADMIN_RESET=1", email)
		}
		existing.PasswordHash = hash
		existing.Status = userDomain.AccountStatusActive
		existing.EmailVerified = true
		existing.Role = userDomain.RoleSuperAdmin
		existing.UpdatedAt = time.Now().UTC()
		if err := users.Update(ctx, existing); err != nil {
			fail("could not update the account: %v", err)
		}
		fmt.Printf("Password reset for %s (super admin). You can sign in now.\n", email)
		return
	}

	user := userDomain.NewUser(email, hash, first, last, userDomain.RoleSuperAdmin)
	user.Status = userDomain.AccountStatusActive
	user.EmailVerified = true
	if err := users.Create(ctx, user); err != nil {
		fail("could not create the account: %v", err)
	}
	fmt.Printf("Created super admin %s. Sign in at your admin address, then change the password to one only you know.\n", email)
}
