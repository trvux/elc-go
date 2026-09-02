// Command seed-admin ensures a super_admin account exists for a given email.
// Ordinary accounts are created automatically (as RoleMember) the first time
// someone signs in via Google or magic link — this is the only way to
// bootstrap the very first account above that, since nothing else can grant
// super_admin. Safe to re-run: if the account already exists, it only
// (re)promotes its role, and is also the break-glass path if every admin
// account somehow got demoted or disabled.
//
// Usage:
//
//	ADMIN_EMAIL=you@example.com ADMIN_NAME='Bảo Huy' go run ./cmd/seed-admin
//
// or `make seed-admin` with the same variables exported first.
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

	email := requireEnv("ADMIN_EMAIL")
	name := os.Getenv("ADMIN_NAME")

	ctx := context.Background()
	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	repo := authinfra.NewPostgresUserRepository(pool)

	existing, err := repo.GetByEmail(ctx, email)
	if err != nil {
		fatalf("look up existing user: %v", err)
	}

	if existing == nil {
		user, err := authdomain.NewOAuthUser(email, name, "", nil, authdomain.RoleSuperAdmin)
		if err != nil {
			fatalf("build user: %v", err)
		}
		created, err := repo.Create(ctx, user)
		if err != nil {
			fatalf("create user: %v", err)
		}
		fmt.Printf("created super_admin %q (%s)\n", created.Email(), created.ID())
		return
	}

	existing.SetRole(authdomain.RoleSuperAdmin)
	updated, err := repo.Update(ctx, existing)
	if err != nil {
		fatalf("update user: %v", err)
	}
	fmt.Printf("promoted existing account %q (%s) to super_admin\n", updated.Email(), updated.ID())
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("%s is required", key)
	}
	return v
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
