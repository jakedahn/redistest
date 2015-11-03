package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/garyburd/redigo/redis"
)

const (
	secretKey = "zomgwtfbbq"
)

var (
	redisHost string
	redisConn redis.Conn
	tokens    []string
)

type RedisClient struct {
	redis_hostname      string
	redisConnectionPool redis.Pool
}

func generateToken(emailPrefix string) string {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["email"] = fmt.Sprintf("jake+%s@planet.com", emailPrefix)
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
		MaxIdle:     3,
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

func test_time_for_direct_redis_gets() {
	jobs := make(chan string)
	workercount := 80
	results := make(chan time.Duration)
	durations := []time.Duration{}

	// queue jobs
	go func() {
		for i := 0; i < 10000; i++ {
			jobs <- tokens[i]
		}
	}()

	for i := 0; i < workercount; i++ {
		go worker(i, jobs, results)
	}

	for i := 0; i < 10000; i++ {
		durations = append(durations, <-results)
	}

	fmt.Println(durations)
	total := time.Second * 0
	for i := 0; i < 10000; i++ {
		total += durations[i]
	}

	avg := total / 10000

	fmt.Println("########################")
	fmt.Println("Total Time:", total)
	fmt.Println("Avg Time:", avg)
	fmt.Println("########################")
}

func test_http() {
	jobs := make(chan string)
	workercount := 1
	results := make(chan time.Duration)
	durations := []time.Duration{}

	// queue jobs
	go func() {
		for i := 0; i < 10000; i++ {
			jobs <- tokens[i]
		}
	}()

	for i := 0; i < workercount; i++ {
		go httpworker(i, jobs, results)
	}

	for i := 0; i < 10000; i++ {
		durations = append(durations, <-results)
	}

	fmt.Println(durations)
	total := time.Second * 0
	for i := 0; i < 10000; i++ {
		total += durations[i]
	}

	avg := total / 10000

	fmt.Println("########################")
	fmt.Println("Total Time:", total)
	fmt.Println("Avg Time:", avg)
	fmt.Println("########################")
}

func worker(i int, jobs chan string, results chan time.Duration) {
	for token := range jobs {
		fmt.Printf("worker %v: job %v\n\n", i, token)

		start := time.Now()
		redisConn.Do("GET", token)
		elapsed := time.Since(start)

		results <- elapsed
	}
}

func httpworker(i int, jobs chan string, results chan time.Duration) {
	for token := range jobs {
		fmt.Printf("worker %v: job %v\n\n", i, token)

		start := time.Now()
		apiUrl := "http://looce.com:8080/"
		data := url.Values{}
		data.Set("token", token)

		resp, err := http.PostForm(apiUrl, data)
		resp.Body.Close()
		if err != nil {
			panic(err)
		}
		elapsed := time.Since(start)

		results <- elapsed
	}
}

func main() {

	flag.StringVar(&redisHost, "redis-host", "localhost", "Specify the redis hostname")
	flag.Parse()

	var err error
	rc, err := NewRedisClient(redisHost)
	if err != nil {
		fmt.Println("########")
		panic(err)
		fmt.Println("########")
	}
	redisConn = rc.redisConnectionPool.Get()

	Setup()
	// test_time_for_direct_redis_gets()
	test_http()
}
