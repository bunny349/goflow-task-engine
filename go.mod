module github.com/YOUR_USERNAME/taskqueue

go 1.22.2

require (
	github.com/alicebob/miniredis/v2 v2.32.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.5.1
)

require (
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
)

// Note: this replace directive routes an indirect dependency (used by
// miniredis, test-only) through its GitHub mirror instead of the default
// golang.org proxy. It's not required for normal Go module resolution —
// included here only because it was built in a network-restricted
// environment. Safe to remove if you hit any issues; `go mod tidy` will
// re-resolve it normally on a standard connection.
replace golang.org/x/sys => github.com/golang/sys v0.20.0
