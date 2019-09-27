package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bmizerany/perks/quantile"
	"github.com/go-redis/redis"
)

func errHndlr(err error) {
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func worker(id int, jobs <-chan int, results chan<- time.Duration, hostname string, port int, password string) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", hostname, port),
		Password: password, // no password set
	})
	for j := range jobs {
		t1 := time.Now()
		pipe := client.Pipeline()
		pipe.Set(fmt.Sprintf("DTMDTM:%d", j), strconv.Itoa(j), 0)
		pipe.Del(strconv.Itoa(j))
		pipe.Exec()
		results <- time.Since(t1)
	}
	client.Close()
}

func main() {
	redisHost := flag.String("host", "localhost", "Redis Host")
	redisPort := flag.Int("port", 6379, "Redis Port")
	showDetails := flag.Bool("details", false, "Disable each line result")
	showSummary := flag.Bool("summary", false, "Print summary stats")
	redisPassword := flag.String("password", "", "RedisPassword")
	messageCount := flag.Int("message_count", 10000, "run this man times")
	threadCount := flag.Int("threadcount", 10, "run this man threads")
	flag.Parse()

	q := quantile.NewTargeted(0.90, 0.95, 0.99)

	jobs := make(chan int, *messageCount)
	results := make(chan time.Duration, *messageCount)

	for w := 0; w <= *threadCount; w++ {
		go worker(w, jobs, results, *redisHost, *redisPort, *redisPassword)
	}

	for j := 0; j <= *messageCount-1; j++ {
		jobs <- j
	}
	close(jobs)

	// Finally we collect all the results of the work.
	for a := 0; a <= *messageCount-1; a++ {
		v := <-results
		if *showDetails {
		} else {
			fmt.Println(a, v.Milliseconds())
		}
		if a >= *threadCount {
			q.Insert(float64(v.Milliseconds()))
		}
	}

	if *showSummary {
		fmt.Printf("---------------- Summary -----------------------\n")
		fmt.Println("perc90:", q.Query(0.90))
		fmt.Println("perc95:", q.Query(0.95))
		fmt.Println("perc99:", q.Query(0.99))
		fmt.Println("count:", q.Count())
		fmt.Printf("------------------------------------------------\n")
	}
}
