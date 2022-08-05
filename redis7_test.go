package xcache_test

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/actforgood/xcache"
	"github.com/actforgood/xconf"
)

func init() {
	var _ xcache.Cache = (*xcache.Redis7)(nil) // test Redis7 is a Cache
}

func ExampleRedis7() {
	cache := xcache.NewRedis7(xcache.RedisConfig{
		Addrs: []string{"127.0.0.1:6379"},
	})

	ctx := context.Background()
	key := "example-redis"
	value := []byte("Hello Redis Cache")
	ttl := 10 * time.Minute

	// save a key for 10 minutes
	if err := cache.Save(ctx, key, value, ttl); err != nil {
		fmt.Println(err)
	}

	// load the key's value
	if value, err := cache.Load(ctx, key); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(value))
	}

	// should output:
	// Hello Redis Cache
}

func ExampleRedis7_withXConf() {
	// Setup the config our application will use (here used a NewFlattenLoader over a json source)
	// You can use whatever config loader suits you as long as needed xcache keys are present.
	config, err := xconf.NewDefaultConfig(
		xconf.NewFlattenLoader(xconf.JSONReaderLoader(bytes.NewReader([]byte(`{
			"xcache": {
			  "redis": {
				"addrs": [
				  "127..0.0.1:6379"
				],
				"db": 0,
				"auth": {
				  "username": "",
				  "password": ""
				},
				"timeout": {
				  "dial": "5s",
				  "read": "6s",
				  "write": "10s"
				},
				"cluster": {
				  "readonly": true
				},
				"failover": {
				  "mastername": "mymaster",
				  "auth": {
					"username": "",
					"password": ""
				  }
				}
			  }
			}
		  }`)))),
		xconf.DefaultConfigWithReloadInterval(5*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer config.Close()

	// Initialize the cache our application will use.
	cache := xcache.NewRedis7WithConfig(config)
	defer cache.Close()

	// From this point forward you can do whatever you want with the cache.
	// Any config that gets changed, cache will reconfigure itself in a time up to reload interval (5 mins here)
	// without the need of restarting/redeploying our application.
	// For example, let's assume we notice a lot of timeout errors, until we figure it out what's happening with our Redis server,
	// we can increase read/write timeouts.
}
