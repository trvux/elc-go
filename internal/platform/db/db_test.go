package db

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const testDSN = "postgres://u:p@127.0.0.1:5432/db"

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestPoolConfig_DefaultsToTwelve(t *testing.T) {
	cfg, err := poolConfig(testDSN, env(nil))
	if err != nil {
		t.Fatal(err)
	}
	// A literal on purpose: comparing against defaultMaxConns would still pass
	// if someone changed the constant back to pgx's starving default of 4.
	if cfg.MaxConns != 12 {
		t.Fatalf("MaxConns = %d, want 12 (pgx's own default of 4 starved the API)", cfg.MaxConns)
	}
}

func TestPoolConfig_EnvOverridesDefault(t *testing.T) {
	cfg, err := poolConfig(testDSN, env(map[string]string{"DB_MAX_CONNS": "20"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConns != 20 {
		t.Fatalf("MaxConns = %d, want 20", cfg.MaxConns)
	}
}

func TestPoolConfig_DSNParameterWinsOverEnv(t *testing.T) {
	cfg, err := poolConfig(testDSN+"?pool_max_conns=7", env(map[string]string{"DB_MAX_CONNS": "20"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConns != 7 {
		t.Fatalf("MaxConns = %d, want 7 (explicit DSN parameter must win)", cfg.MaxConns)
	}
}

func TestPoolConfig_RejectsBadEnvInsteadOfFallingBack(t *testing.T) {
	for _, bad := range []string{"abc", "0", "-3", "101", "12.5"} {
		_, err := poolConfig(testDSN, env(map[string]string{"DB_MAX_CONNS": bad}))
		if err == nil {
			t.Errorf("DB_MAX_CONNS=%q accepted; a typo must fail loudly", bad)
			continue
		}
		if !strings.Contains(err.Error(), "DB_MAX_CONNS") {
			t.Errorf("error for %q doesn't name the variable: %v", bad, err)
		}
	}
}

func TestPoolConfig_KeepsSimpleProtocol(t *testing.T) {
	cfg, err := poolConfig(testDSN, env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.ConnConfig.DefaultQueryExecMode; got != pgx.QueryExecModeSimpleProtocol {
		t.Fatalf("DefaultQueryExecMode = %v, want simple protocol (behavior preserved from before the pool change)", got)
	}
}
