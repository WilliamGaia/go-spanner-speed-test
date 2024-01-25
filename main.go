package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type Entry struct {
	UUID           string `json:"uuid"`
	Severity       string `json:"severity,omitempty"`
	ReceiveReqTime string `json:"request_received"`
	StartTime      string `json:"api_call_start"`
	EndTime        string `json:"api_call_response"`
	ElapsedTime    string `json:"api_process_elapsed"`
	ScannedRow     string `json:"row_scanned"`
}

// String renders an entry structure to the JSON format expected by Cloud Logging.
func (e Entry) String() string {
	if e.Severity == "" {
		e.Severity = "INFO"
	}
	out, err := json.Marshal(e)
	if err != nil {
		log.Printf("json.Marshal: %v", err)
	}
	return string(out)
}

var loc *time.Location
var client *spanner.Client

func queryWithParameter(w io.Writer, resultChan chan<- string, rowChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	// return nil
	ctx := context.Background()

	queryCases := []struct {
		RegisterFrom int
		StartTime    string
		EndTime      string
		QuerySQL     string
	}{
		{5, "2020-01-31T14:58:21.200Z", "2024-01-31T14:58:31.200Z", "select registerFrom, sum(policyAcceptance) from User_partial where registerFrom=@p1 and registerTime>= @p2 and registerTime<= @p3 group by registerFrom;"},
		{1, "2023-11-28T07:23:16.501888Z", "2023-11-28T07:27:16.501888Z", "select registerFrom, sum(policyAcceptance) from User_partial where registerFrom=@p1 and registerTime>= @p2 and registerTime<= @p3 group by registerFrom;"},
		{2, "2023-11-28T07:23:16.501888Z", "2023-11-28T07:23:29.918698Z", "select registerFrom, sum(policyAcceptance) from User_partial where registerFrom=@p1 and registerTime>= @p2 and registerTime<= @p3 group by registerFrom;"},
	}

	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)

	selectedCase := queryCases[randGen.Intn(len(queryCases))]

	stmt := spanner.Statement{
		SQL: selectedCase.QuerySQL,
		Params: map[string]interface{}{
			"p1": selectedCase.RegisterFrom,
			"p2": selectedCase.StartTime,
			"p3": selectedCase.EndTime,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result strings.Builder
	var rowCounts string

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			resultChan <- fmt.Sprintf("Error: %v", err)
			return
		}
		var registerFrom, policyAcceptance_sum int64
		if err := row.Columns(&registerFrom, &policyAcceptance_sum); err != nil {
			resultChan <- fmt.Sprintf("Error: %v", err)
			return
		}

		switch selectedCase.RegisterFrom {
		case 5:
			rowCounts = "10w"
		case 1:
			rowCounts = "1w"
		case 2:
			rowCounts = "100"
		default:
			rowCounts = "unknown"
		}
		fmt.Fprintf(&result, "registerForm: %d, policyAcceptance_sum: %d\n", registerFrom, policyAcceptance_sum)
	}
	resultChan <- result.String()
	rowChan <- rowCounts
}

func startTest(c *gin.Context) {
	start := time.Now().In(loc)
	resultChan := make(chan string, 1)
	rowChan := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	//timediff for query
	start_query := time.Now().In(loc)
	go queryWithParameter(os.Stdout, resultChan, rowChan, &wg)
	wg.Wait()
	result := <-resultChan
	scanRow := <-rowChan
	if strings.HasPrefix(result, "Error:") {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": result})
		return
	}

	elapsed_query := time.Since(start_query).Microseconds()
	elapsed := time.Since(start).Microseconds()

	log.Println(Entry{
		Severity:       "WARNING",
		UUID:           c.Request.Header["X-Client-Uuid"][0],
		ReceiveReqTime: start.String(),
		StartTime:      start_query.String(),
		EndTime:        fmt.Sprintf("%v", elapsed_query),
		ElapsedTime:    fmt.Sprintf("%v", elapsed),
		ScannedRow:     scanRow,
	})
	defer c.IndentedJSON(http.StatusOK, result)
}

func init() {
	log.SetFlags(0)
	project := os.Getenv("PROJECT")
	instance := os.Getenv("INSTANCE")
	database := os.Getenv("DATABASE")
	if project == "" {
		project = "williamlab"
		instance = "go-spanner-test-instance"
		database = "go-spanner-db"
	}
	databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	loc, _ = time.LoadLocation("Asia/Taipei")
	//create connection object
	ctx := context.Background()
	var err error
	client, err = spanner.NewClient(ctx, databaseName)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s \n", err)
	}
	//handle initial session
	stmt := spanner.Statement{
		SQL: `select 1;`,
	}
	iter := client.Single().Query(ctx, stmt)
	err = iter.Do(func(row *spanner.Row) error {
		if err := row.Columns(nil); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stdout, "Error executing initial query: %v\n", err)
	}
}

func main() {
	router := gin.Default()
	router.GET("/startTest", startTest)
	router.Run(":8080")
}
