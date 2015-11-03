package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/codegangsta/martini"
	"github.com/garyburd/redigo/redis"
	"github.com/martini-contrib/render"
)

var (
	redisAddress   = flag.String("redis-address", ":6379", "Address to the Redis server")
	maxConnections = flag.Int("max-connections", 10, "Max connections to Redis")
)

func main() {
	martini.Env = martini.Prod

	flag.Parse()

	redisPool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", *redisAddress)

		if err != nil {
			return nil, err
		}

		return c, err
	}, *maxConnections)

	defer redisPool.Close()

	m := martini.Classic()

	m.Map(redisPool)

	m.Use(render.Renderer())

	m.Post("/", func(r render.Render, pool *redis.Pool, params martini.Params, req *http.Request) {
		token := strings.TrimSpace(req.FormValue("token"))

		c := pool.Get()
		defer c.Close()

		status, err := redis.Int(c.Do("GET", token))
		if err != nil {
			fmt.Print("-")
			r.JSON(403, map[string]interface{}{"status": "ERR"})
		}
		if status == 1 {
			fmt.Print(".")
			r.JSON(200, map[string]interface{}{"status": "AUTHD"})
		}

	})

	m.Run()
}
