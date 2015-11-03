package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

const (
	secretKey = "zomgwtfbbq"
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

func generateToken(emailPrefix string) string {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["email"] = fmt.Sprintf("jake+%s@foo.com", emailPrefix)
	token.Claims["exp"] = time.Now().Add(time.Minute * 100).Unix()
	token.Claims["groups"] = []string{"sre", "mcdev", "users"}

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		panic(err)
	}
	return tokenString
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

// Here we will make a bunch of tokens, store them in redis
func Setup() {
	tokens = []string{}
	tokenCount := 10000

	for i := 0; i < tokenCount; i++ {
		tokens = append(tokens, generateToken(strconv.Itoa(i)))
	}

	for i := 0; i < tokenCount; i++ {
		token := tokens[i]
		redisConn.Do("SET", token, 1)
	}
	fmt.Print("All of the tokens are now in redis")
}

func main() {
	flag.StringVar(&redisHost, "redis-host", "localhost", "Specify the redis hostname")
	flag.Parse()

	var err error
	rc, err := NewRedisClient(redisHost)
	if err != nil {
		panic(err)
	}
	redisConn = rc.redisConnectionPool.Get()

	Setup()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", Index)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func Index(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	rc, err := NewRedisClient(redisHost)
	if err != nil {
		panic(err)
	}
	status, err := redis.Int(rc.redisConnectionPool.Get().Do("GET", token))
	if err != nil {
		panic(err)
		http.Error(w, "0", 403)
	}
	if status == 1 {
		fmt.Fprintln(w, status)
	}
}
