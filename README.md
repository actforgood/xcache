# Xcache

[![Build Status](https://github.com/actforgood/xcache/actions/workflows/build.yml/badge.svg)](https://github.com/actforgood/xcache/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://raw.githubusercontent.com/actforgood/xcache/main/LICENSE)
[![Coverage Status](https://coveralls.io/repos/github/actforgood/xcache/badge.svg?branch=main)](https://coveralls.io/github/actforgood/xcache?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/actforgood/xcache.svg)](https://pkg.go.dev/github.com/actforgood/xcache)  

---

Package `xcache` offers caching alternatives for an application like a local in memory cache,
or distributed Redis cache, or combination of those two in a multi layered cache.


### Cache adapters
- `Memory` - a local in memory cache, relies upon Freecache package.  
- `Redis6` - Redis version 6 cache (single instance / sentinel failover / cluster).  
- `Redis7` - Redis version 7 cache (single instance / sentinel failover / cluster).  
- `Multi` - A multi layer cache.  
- `Nop` - A no-operation cache.  
- `Mock` - A stub that can be used in Unit Tests.  


### The Cache contract
Looks like:  
```go
// Cache provides prototype a for storing and returning a key-value into/from cache.
type Cache interface {
	// Save stores the given key-value with expiration period into cache.
	// An expiration period equal to 0 (NoExpire) means no expiration.
	// A negative expiration period triggers deletion of key.
	// It returns an error if the key could not be saved.
	Save(ctx context.Context, key string, value []byte, expire time.Duration) error

	// Load returns a key's value from cache, or an error if something bad happened.
	// If the key is not found, ErrNotFound is returned.
	Load(ctx context.Context, key string) ([]byte, error)

	// TTL returns a key's remaining time to live, or an error if something bad happened.
	// If the key is not found, a negative TTL is returned.
	// If the key has no expiration, 0 (NoExpire) is returned.
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Stats returns some statistics about cache's memory/keys.
	// It returns an error if something goes wrong.
	Stats(context.Context) (Stats, error)
}
```

### Examples
###### Memory
```go
func ExampleMemory() {
	cache := xcache.NewMemory(10 * 1024 * 1024) // 10 Mb

	ctx := context.Background()
	key := "example-memory"
	value := []byte("Hello Memory Cache")
	ttl := 10 * time.Minute

	// save a key for 10 minutes
	if err := cache.Save(ctx, key, value, ttl); err != nil {
		fmt.Println("could not save Memory cache key: " + err.Error())
	}

	// load the key's value
	if value, err := cache.Load(ctx, key); err != nil {
		fmt.Println("could not get Memory cache key: " + err.Error())
	} else {
		fmt.Println(string(value))
	}

	// Output:
	// Hello Memory Cache
}
```
Benchmarks
```shell
go test -run=^# -benchmem -benchtime=5s -bench BenchmarkMemory github.com/actforgood/xcache
goos: linux
goarch: amd64
pkg: github.com/actforgood/xcache
cpu: Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz
BenchmarkMemory_Save-4                  11463078               492.5 ns/op             2 B/op          0 allocs/op
BenchmarkMemory_Save_parallel-4         24244593               261.4 ns/op            25 B/op          1 allocs/op
BenchmarkMemory_Load-4                  30390457               194.0 ns/op            40 B/op          2 allocs/op
BenchmarkMemory_Load_parallel-4         25545855               220.0 ns/op            40 B/op          2 allocs/op
BenchmarkMemory_TTL-4                   55357045               110.5 ns/op             0 B/op          0 allocs/op
BenchmarkMemory_TTL_parallel-4          40464970               153.2 ns/op             0 B/op          0 allocs/op
BenchmarkMemory_Stats-4                  5760609               983.7 ns/op             0 B/op          0 allocs/op
BenchmarkMemory_Stats_parallel-4        23939924               254.5 ns/op             0 B/op          0 allocs/op
```

###### Redis
```go
func ExampleRedis() {
	cache := xcache.NewRedis6(xcache.RedisConfig{ // or xcache.NewRedis7 if you're using ver. 7
		Addrs: []string{"127.0.0.1:6379"},
	})

	ctx := context.Background()
	key := "example-redis"
	value := []byte("Hello Redis Cache")
	ttl := 10 * time.Minute

	// save a key for 10 minutes
	if err := cache.Save(ctx, key, value, ttl); err != nil {
		fmt.Println("could not save Redis cache key: " + err.Error())
	}

	// load the key's value
	if value, err := cache.Load(ctx, key); err != nil {
		fmt.Println("could not get Redis cache key: " + err.Error())
	} else {
		fmt.Println(string(value))
	}

	// close the cache when no needed anymore/at your application shutdown.
	if err := cache.Close(); err != nil {
		fmt.Println("could not close Redis cache: " + err.Error())
	}

	// should output:
	// Hello Redis Cache
}
```
Benchmarks
```shell
go test -tags=integration -run=^# -benchmem -benchtime=5s -bench BenchmarkRedis github.com/actforgood/xcache
goos: linux
goarch: amd64
pkg: github.com/actforgood/xcache
cpu: Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz
BenchmarkRedis6_Save_integration-4                 32479            161610 ns/op             272 B/op          7 allocs/op
BenchmarkRedis6_Save_parallel_integration-4       130350             43058 ns/op             296 B/op          8 allocs/op
BenchmarkRedis6_Load_integration-4                 39960            144686 ns/op             208 B/op          6 allocs/op
BenchmarkRedis6_Load_parallel_integration-4       103783             62320 ns/op             208 B/op          6 allocs/op
BenchmarkRedis6_TTL_integration-4                  43336            158656 ns/op             196 B/op          5 allocs/op
BenchmarkRedis6_TTL_parallel_integration-4        135324             43266 ns/op             196 B/op          5 allocs/op
BenchmarkRedis6_Stats-4                            23257            244013 ns/op            5052 B/op          6 allocs/op
BenchmarkRedis6_Stats_parallel-4                   63873             90896 ns/op            5052 B/op          6 allocs/op
BenchmarkRedis7_Save_integration-4                 38620            162062 ns/op             312 B/op         10 allocs/op
BenchmarkRedis7_Save_parallel_integration-4       129525             46068 ns/op             336 B/op         11 allocs/op
BenchmarkRedis7_Load_integration-4                 42074            153150 ns/op             248 B/op          9 allocs/op
BenchmarkRedis7_Load_parallel_integration-4       139232             43403 ns/op             248 B/op          9 allocs/op
BenchmarkRedis7_TTL_integration-4                  32029            163338 ns/op             236 B/op          8 allocs/op
BenchmarkRedis7_TTL_parallel_integration-4        117226             56544 ns/op             236 B/op          8 allocs/op
BenchmarkRedis7_Stats-4                            23600            254668 ns/op            5604 B/op          9 allocs/op
BenchmarkRedis7_Stats_parallel-4                   59908            100755 ns/op            5604 B/op          9 allocs/op
```

###### Multi
```go
func ExampleMulti() {
	// create a frontend - backend multi cache.
	frontCache := xcache.NewMemory(10 * 1024 * 1024) // 10 Mb
	backCache := xcache.NewRedis6(xcache.RedisConfig{
		Addrs: []string{"127.0.0.1:6379"},
	})
	defer backCache.Close()
	cache := xcache.NewMulti(frontCache, backCache)

	ctx := context.Background()
	key := "example-multi"
	value := []byte("Hello Multi Cache")
	ttl := 10 * time.Minute

	// save a key for 10 minutes
	if err := cache.Save(ctx, key, value, ttl); err != nil {
		fmt.Println("could not save Multi cache key: " + err.Error())
	}

	// load the key's value
	if value, err := cache.Load(ctx, key); err != nil {
		fmt.Println("could not get Multi cache key: " + err.Error())
	} else {
		fmt.Println(string(value))
	}

	// should output:
	// Hello Multi Cache
}
```


### Reconfiguring on the fly the caches
If you need to change caches' configs without redeploying your application, you can use the [xconf](https://github.com/actforgood/xconf) pkg adapter to initialize the caches: `NewMemoryWithConfig` / `NewRedis6WithConfig` / `NewRedis7WithConfig`.


### Monitoring your cache stats
If you need to monitor your cache's statistics, you can check `StatsWatcher` which can help you in this matter. It executes periodically a provided callback upon cache's `Stats`, thus, you can log them / sent them to a metrics system.


### Running tests / benchmarks
in `scripts` folder there is a shell script that sets up a Redis docker based environment with desired configuration and runs integration tests / benchmarks.
```bash
cd /path/to/xcache
./scripts/run_local.sh cluster  // example of running tests in Redis cluster setup
./scripts/run_local.sh single bench // example of running benchmarks in Redis single instance setup.
```

### TODOs:
Things that can be added to pkg, extended:  

- Support also Memcached.

### License
This package is released under a MIT license. See [LICENSE](LICENSE).  
Other 3rd party packages directly used by this package are released under their own licenses.  

* github.com/coocood/freecache - [MIT License](https://github.com/coocood/freecache/blob/master/LICENSE)  
* github.com/go-redis/redis/v8 - [BSD (2 Clause) License](https://github.com/go-redis/redis/blob/v8.11.5/LICENSE)
* github.com/go-redis/redis/v9 - [BSD (2 Clause) License](https://github.com/go-redis/redis/blob/v9.0.0-beta.2/LICENSE)    
* github.com/actforgood/xerr - [MIT License](https://github.com/actforgood/xerr/blob/main/LICENSE)  
* github.com/actforgood/xlog - [MIT License](https://github.com/actforgood/xlog/blob/main/LICENSE)  
* github.com/actforgood/xconf - [MIT License](https://github.com/actforgood/xconf/blob/main/LICENSE)  
