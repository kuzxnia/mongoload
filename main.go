package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// type args struct {
// 	numReqs  uint64
// 	duration time.Duration

// 	numConns uint64

// 	timeout time.Duration
// }
// rateLimit

// batchSize

type Book struct {
	Title  string
	Author string
	ISBN   string
}

type mongoload struct {
	db *mongo.Client
	wg sync.WaitGroup

	rateLimit int

	// rps
	rlp   sync.Mutex
	reqs  int64
	start time.Time
}

func (*mongoload) newMongoLoad() (*mongoload, error) {
	mongoLoad := new(mongoload)

	return mongoLoad, nil
}
func (*mongoload) performSingleWrite()             {}
func (*mongoload) performBatchWrite(batchSize int) {}
func (*mongoload) performSingleRead()              {}
func (*mongoload) worker()                         {}

func main() {
	// iserts
	connectionsAmount := 400
	insertsAmount := 125 * 800

	start := time.Now()

  uri := "mongodb://localhost:27017"
	opts := options.Client().
		ApplyURI(uri).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetAppName("test").
		SetMaxPoolSize(uint64(connectionsAmount * 8)).
		SetMaxConnecting(100).
		SetMaxConnIdleTime(time.Microsecond * 100000)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	if err != nil {
		panic(err)
	} else {
		fmt.Println("connected")
	}

	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		fmt.Println("error in ping to mongo")
	} else {
		fmt.Println("no errors found")
	}

	var wg sync.WaitGroup
	wg.Add(connectionsAmount)

	jobChannel := make(chan int)
	resultChannel := make(chan bool, insertsAmount)

	fmt.Println("worker starting")
	// start workers
	for i := 0; i < connectionsAmount; i++ {
		go worker(&wg, client, jobChannel, resultChannel)
	}
	fmt.Println("after worker starting")

	fmt.Println("main sending jobs")
	// send jobs
	for i := 0; i < insertsAmount; i++ {
		jobChannel <- i
	}
	fmt.Println("main jobs sended")

	close(jobChannel)
	wg.Wait()
	close(resultChannel)

	fmt.Println("done")

	// for result := range resultChannel {
	// 	fmt.Printf("%v \n", result)
	// }

	elapsed := time.Since(start)
	fmt.Printf("Find documents took %s", elapsed)
}

func worker(wg *sync.WaitGroup, client *mongo.Client, jobChannel <-chan int, resultChannel chan bool) {
	defer wg.Done()

	collection := client.Database("test").Collection("go")

	books := []interface{}{
		Book{Title: "The Trial", Author: "Franz Kafka", ISBN: "978-0307595119"},
		Book{Title: "The Castle", Author: "Franz Kafka", ISBN: "978-0307474670"},
		Book{Title: "The Trial", Author: "Franz Kafka", ISBN: "978-0307595119"},
		Book{Title: "The Castle", Author: "Franz Kafka", ISBN: "978-0307474670"},
		Book{Title: "The Hunger Games", Author: "Suzanne Collins", ISBN: "978-0439023481"},
		Book{Title: "Catching Fire", Author: "Suzanne Collins", ISBN: "978-0439023498"},
		Book{Title: "The Trial", Author: "Franz Kafka", ISBN: "978-0307595119"},
		Book{Title: "The Castle", Author: "Franz Kafka", ISBN: "978-0307474670"},
		Book{Title: "The Hunger Games", Author: "Suzanne Collins", ISBN: "978-0439023481"},
		Book{Title: "Catching Fire", Author: "Suzanne Collins", ISBN: "978-0439023498"},
	}

	fmt.Println("worker sending jobs")
	for i := range jobChannel {
		resultChannel <- insertManyDocuments(collection, books, i)
	}
}

func insertManyDocuments(collection *mongo.Collection, books []interface{}, i int) bool {
	_, err := collection.InsertMany(context.Background(), books)
	// if err != nil {
	// 	fmt.Printf("error %v \n", err)
	// } else {
	// 	fmt.Printf("inserted %d \n", i)
	// }
	return bool(err == nil)
}

func readDocuments(collection *mongo.Collection) bool {
	// start := time.Now()
	batch_size := int32(1000)

	cursor, err := collection.Find(context.Background(), bson.M{"author": "Franz Kafkaaa"}, &options.FindOptions{BatchSize: &batch_size})
	if err != nil {
		log.Fatal(err)
	}

	defer cursor.Close(context.Background())

	// var results []Book
	// if err = cursor.All(context.Background(), &results); err != nil {
	// 	panic(err)
	// }

	//  println(len(results))

	// for _, book := range results {
	// 	res, _ := json.Marshal(book)
	// 	fmt.Println(string(res))
	// }

	totalFound := 0
	for cursor.Next(context.Background()) {
		var book Book

		if err = cursor.Decode(&book); err != nil {
			log.Fatal(err)
		}
		totalFound++
	}

	// elapsed := time.Since(start)
	// fmt.Printf("Find documents took %s", elapsed)
	return true
}