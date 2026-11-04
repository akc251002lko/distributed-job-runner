package config
import("testing";"time")
func clearConfigEnv(t *testing.T){t.Helper();for _,k:=range []string{"DATABASE_URL","API_DATABASE_URL","WORKER_DATABASE_URL","APP_ENV","API_TOKEN","API_KEYS","POLL_INTERVAL","LEASE_DURATION","WORKER_CONCURRENCY","API_RATE_LIMIT_RPS","API_RATE_LIMIT_BURST"}{t.Setenv(k,"")}}
func TestLoadDefaults(t *testing.T){clearConfigEnv(t);t.Setenv("DATABASE_URL","postgres://localhost/djr");c,err:=Load();if err!=nil{t.Fatalf("Load(): %v",err)};if c.PollInterval!=time.Second||c.LeaseDuration!=30*time.Second||c.WorkerConcurrency!=4{t.Fatalf("unexpected defaults: %+v",c)}}
func TestProductionRequiresSeparatedDatabaseAndStrongAuth(t *testing.T){clearConfigEnv(t);t.Setenv("APP_ENV","Production");t.Setenv("API_DATABASE_URL","postgres://api@localhost/djr?sslmode=verify-full");t.Setenv("WORKER_DATABASE_URL","postgres://worker@localhost/djr?sslmode=verify-full");t.Setenv("API_TOKEN","0123456789abcdef0123456789abcdef");if _,err:=Load();err!=nil{t.Fatalf("valid config rejected: %v",err)};t.Setenv("DATABASE_URL","postgres://owner@localhost/djr");if _,err:=Load();err==nil{t.Fatal("shared production database URL accepted")}}

func TestProductionRequiresEncryptedDatabaseConnections(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_DATABASE_URL", "postgres://api@localhost/djr?sslmode=disable")
	t.Setenv("WORKER_DATABASE_URL", "postgres://worker@localhost/djr?sslmode=verify-full")
	t.Setenv("API_TOKEN", "0123456789abcdef0123456789abcdef")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted a database connection without TLS")
	}
}

func TestProductionRequiresDistinctDatabaseRoles(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_DATABASE_URL", "postgres://djr_app@localhost/djr?sslmode=require")
	t.Setenv("WORKER_DATABASE_URL", "postgres://djr_app@localhost/djr?sslmode=require")
	t.Setenv("API_TOKEN", "0123456789abcdef0123456789abcdef")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted a shared API/worker database role")
	}
}

func TestProductionAcceptsKeywordConnectionStrings(t *testing.T) {
	if err := validateProductionDatabaseURLs(
		"host=db.example user=api dbname=djr sslmode=verify-full",
		"host=db.example user=worker dbname=djr sslmode=require",
	); err != nil {
		t.Fatalf("valid keyword connection strings rejected: %v", err)
	}
}

func TestAPIAuthenticationModesAreMutuallyExclusive(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/djr")
	t.Setenv("API_TOKEN", "0123456789abcdef0123456789abcdef")
	t.Setenv("API_KEYS", "mobile=abcdef0123456789abcdef0123456789")
	if _, err := Load(); err == nil {
		t.Fatal("both API_TOKEN and API_KEYS were accepted")
	}
}

func TestAPIKeyValidation(t *testing.T){clearConfigEnv(t);t.Setenv("DATABASE_URL","postgres://localhost/djr");t.Setenv("API_KEYS","mobile=short");if _,err:=Load();err==nil{t.Fatal("weak key accepted")};t.Setenv("API_KEYS","mobile=0123456789abcdef0123456789abcdef");if _,err:=Load();err!=nil{t.Fatalf("valid key rejected: %v",err)}}
func TestLeaseAndConcurrencyValidation(t *testing.T){clearConfigEnv(t);t.Setenv("DATABASE_URL","postgres://localhost/djr");t.Setenv("POLL_INTERVAL","10s");t.Setenv("LEASE_DURATION","20s");if _,err:=Load();err==nil{t.Fatal("short lease accepted")};t.Setenv("POLL_INTERVAL","1s");t.Setenv("LEASE_DURATION","30s");t.Setenv("WORKER_CONCURRENCY","0");if _,err:=Load();err==nil{t.Fatal("zero concurrency accepted")}}

func TestAPIKeyClientIDsAreValidated(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL","postgres://localhost/djr")
	t.Setenv("API_KEYS","bad id=0123456789abcdef0123456789abcdef")
	if _,err:=Load();err==nil{t.Fatal("invalid client ID accepted")}
	t.Setenv("API_KEYS","mobile=0123456789abcdef0123456789abcdef,mobile=abcdef0123456789abcdef0123456789")
	if _,err:=Load();err==nil{t.Fatal("duplicate client ID accepted")}
}

func TestAPIKeyTokensMustBeUnique(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/djr")
	token := "0123456789abcdef0123456789abcdef"
	t.Setenv("API_KEYS", "client-a="+token+",client-b="+token)
	if _, err := Load(); err == nil {
		t.Fatal("duplicate bearer token assigned to multiple clients was accepted")
	}
}

func TestRateLimitRejectsNonFiniteValues(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf", "-Inf"} {
		t.Run(value, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv("DATABASE_URL", "postgres://localhost/djr")
			t.Setenv("API_RATE_LIMIT_RPS", value)
			if _, err := Load(); err == nil {
				t.Fatalf("non-finite rate limit %q was accepted", value)
			}
		})
	}
}

func TestPollIntervalCannotOverflowLeaseValidation(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/djr")
	t.Setenv("POLL_INTERVAL", "2000000000h")
	if _, err := Load(); err == nil {
		t.Fatal("overflowing poll interval was accepted")
	}
}
