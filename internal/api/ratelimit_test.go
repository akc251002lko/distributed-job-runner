package api
import "testing"
func TestRateLimiterEnforcesBurstAndIsolatesClients(t *testing.T){l:=newRateLimiter(100,2);if !l.allow("a")||!l.allow("a"){t.Fatal("initial burst rejected")};if l.allow("a"){t.Fatal("request beyond burst accepted")};if !l.allow("b"){t.Fatal("client buckets are not isolated")}}
