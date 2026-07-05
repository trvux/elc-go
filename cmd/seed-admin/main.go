// Command seed-admin creates or resets the password of a single admin
// account, taking every value from the environment so a real person's
// name/email/phone/password is never written into a committed migration or
// any other file that ends up in git history.
//
// Usage:
//
//	ADMIN_USERNAME=tranvux ADMIN_EMAIL=you@example.com ADMIN_PASSWORD='...' \
//	ADMIN_NAME='Bảo Huy' ADMIN_PHONE=0909411633 ADMIN_ROLE=super_admin \
//	  go run ./cmd/seed-admin
//
// or `make seed-admin` with the same variables exported first. If the email
// already exists, only its password is reset — this is also the intended
// break-glass path if the bootstrap account is ever locked out.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	authinfra "github.com/trvux/elc-go/internal/auth/infrastructure"
	"github.com/trvux/elc-go/internal/platform/db"
)

func main() {
	_ = godotenv.Load()

	username := requireEnv("ADMIN_USERNAME")
	email := requireEnv("ADMIN_EMAIL")
	password := requireEnv("ADMIN_PASSWORD")
	name := os.Getenv("ADMIN_NAME")
	phone := os.Getenv("ADMIN_PHONE")
	role := authdomain.Role(envOr("ADMIN_ROLE", string(authdomain.RoleSuperAdmin)))

	ctx := context.Background()
	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	hasher := authinfra.NewBcryptPasswordHasher()
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		fatalf("hash password: %v", err)
	}

	repo := authinfra.NewPostgresUserRepository(pool)

	existing, err := repo.GetByEmail(ctx, email)
	if err != nil {
		fatalf("look up existing user: %v", err)
	}

	if existing == nil {
		user, err := authdomain.NewUser(username, email, passwordHash, name, phone, role)
		if err != nil {
			fatalf("build user: %v", err)
		}
		created, err := repo.Create(ctx, user)
		if err != nil {
			fatalf("create user: %v", err)
		}
		fmt.Printf("created admin %q (%s), role=%s\n", created.Username(), created.ID(), created.Role())
		return
	}

	// Only resets the password on an existing account — deliberately narrow,
	// this tool is for bootstrap/break-glass, not general profile editing.
	existing.SetPasswordHash(passwordHash)
	updated, err := repo.Update(ctx, existing)
	if err != nil {
		fatalf("update user: %v", err)
	}
	fmt.Printf("reset password for existing admin %q (%s)\n", updated.Username(), updated.ID())
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("%s is required", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
