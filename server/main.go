package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

var (
	redisHost string
	redisConn redis.Conn
	tokens    []string
	rc        RedisClient
)

type RedisClient struct {
	redis_hostname      string
	redisConnectionPool redis.Pool
}

func newRedisPool(redisHostname string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     1000,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:6379", redisHostname))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func NewRedisClient(redisHostname string) (*RedisClient, error) {
	c := &RedisClient{redis_hostname: redisHostname}
	c.redisConnectionPool = *newRedisPool(redisHostname)
	c.redisConnectionPool.Get().Do("PING")
	if c.redisConnectionPool.ActiveCount() == 0 {
		return nil, fmt.Errorf("Unable to connect to Redis")
	}
	return c, nil
}

func Index(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.FormValue("token"))
	rc, err := NewRedisClient(redisHost)
	if err != nil {
		panic(err)
	}

	status, err := redis.Int(rc.redisConnectionPool.Get().Do("GET", token))
	if err != nil {
		fmt.Print("-")
		http.Error(w, "0", 403)
	}
	if status == 1 {
		fmt.Print(".")
		fmt.Fprintln(w, status)
	}
}

func main() {
	flag.StringVar(&redisHost, "redis-host", "localhost", "Specify the redis hostname")
	flag.Parse()

	rc, err := NewRedisClient(redisHost)
	if err != nil {
		panic(err)
	}
	redisConn = rc.redisConnectionPool.Get()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", Index)
	log.Fatal(http.ListenAndServe(":8080", router))
}
