// Admin CLI — currently exposes `create-admin` only. See docs/25-admin-accounts.md.
//
// Usage:
//   go run ./cmd/admin create-admin --email admin@example.com
//   (password is read from stdin, never from argv)
//
// Admin accounts CANNOT be created through any HTTP endpoint. This binary
// requires DB credentials via the same config loader as the API server.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/config"
	"github.com/dauxuanhoanghung/url-shortener/internal/database"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		usageAndExit()
	}
	switch os.Args[1] {
	case "create-admin":
		if err := cmdCreateAdmin(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usageAndExit()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		usageAndExit()
	}
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage: admin <command> [flags]")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  create-admin --email <email>    Create a new admin account (password via stdin)")
	os.Exit(2)
}

func cmdCreateAdmin(args []string) error {
	fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
	email := fs.String("email", "", "admin email address")
	_ = fs.Parse(args)

	if *email == "" {
		return errors.New("--email is required")
	}
	if !strings.Contains(*email, "@") {
		return errors.New("email looks invalid")
	}

	password, err := readPassword()
	if err != nil {
		return fmt.Errorf("reading password: %w", err)
	}
	if len(password) < 12 {
		return errors.New("admin password must be at least 12 characters")
	}
	if !containsDigit(password) || !containsSymbol(password) {
		return errors.New("admin password must contain at least one digit and one symbol")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	userRepo     := repository.NewUserRepository(pool)
	userPlanRepo := repository.NewUserPlanRepository(pool)
	auditRepo    := repository.NewAdminAuditRepository(pool)

	if existing, _ := userRepo.GetByEmail(ctx, *email); existing != nil {
		return fmt.Errorf("an account with email %s already exists", *email)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now()
	// Admin accounts are born verified — operator creating the account
	// is the ground truth of ownership, no signup email needed.
	verified := now
	admin, err := userRepo.Create(ctx, &model.User{
		ID:              uuid.New(),
		Email:           *email,
		PasswordHash:    string(hashed),
		Role:            model.RoleAdmin,
		EmailVerifiedAt: &verified,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return fmt.Errorf("inserting admin: %w", err)
	}

	// Admin accounts also get a user_plans row (free tier) so any code
	// that reads user_plans unconditionally won't error.
	if _, err := userPlanRepo.Create(ctx, admin.ID, "free"); err != nil {
		return fmt.Errorf("creating user_plans for admin: %w", err)
	}

	after, _ := json.Marshal(map[string]string{"email": admin.Email, "role": admin.Role})
	if err := auditRepo.Create(ctx, &repository.AdminAuditEntry{
		ID:         uuid.New(),
		ActorID:    nil, // CLI-originated, no authenticated admin
		Action:     "create_admin",
		TargetType: "user",
		TargetID:   admin.ID.String(),
		After:      after,
		CreatedAt:  now,
	}); err != nil {
		// Admin exists but audit failed — surface the problem but don't
		// pretend the create didn't happen.
		fmt.Fprintf(os.Stderr, "warning: admin created but audit log failed: %v\n", err)
	}

	fmt.Printf("created admin: id=%s email=%s\n", admin.ID, admin.Email)
	return nil
}

// readPassword: stdin-only. Never from argv (which shows up in `ps`).
// Uses a TTY prompt when interactive, plain line-read when piped.
func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Print("password: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSymbol(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		default:
			return true
		}
	}
	return false
}
