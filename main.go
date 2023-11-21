package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func queryWithParameter(w io.Writer, client *spanner.Client) error {
	ctx := context.Background()

	stmt := spanner.Statement{
		SQL: `select registerFrom, sum(policyAcceptance) as policyAcceptance_sum
					  from User 
					  where registerFrom=@p1
					  and registerTime>= @p2 and registerTime<= @p3
					  group by registerFrom;`,
		Params: map[string]interface{}{
			"p1": 5,
			"p2": "2020-01-31T14:58:21.200Z",
			"p3": "2020-01-31T14:58:31.200Z",
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var registerFrom, policyAcceptance_sum int64
		if err := row.Columns(&registerFrom, &policyAcceptance_sum); err != nil {
			return err
		}
		fmt.Fprintf(w, "%d %d\n", registerFrom, policyAcceptance_sum)
	}

}

func startTest(c *gin.Context) {
	// project := os.Getenv("PROJECT")
	// instance := os.Getenv("INSTANCE")
	// database := os.Getenv("DATABASE")
	loc, _ := time.LoadLocation("Asia/Taipei")
	start := time.Now().In(loc)

	// //create connection object
	// databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	// ctx := context.Background()
	// client, err := spanner.NewClient(ctx, databaseName)
	// if err != nil {
	// 	fmt.Fprintf(os.Stdout, "%s \n", err)
	// }
	// defer client.Close()

	//timediff for query
	start_query := time.Now().In(loc)
	// queryWithParameter(os.Stdout, client)
	elapsed_query := time.Since(start_query).Microseconds()
	elapsed := time.Since(start).Microseconds()
	// fmt.Fprintf(
	// 	os.Stdout,
	// 	"start: %s, totaltime: %s, querytime: %s\n",
	// 	start.Format(time.RFC3339),
	// 	elapsed,
	// 	elapsed_query)
	fmt.Println(time.Since(start).String())
	log.Println(Entry{
		Severity:       "WARNING",
		UUID:           c.Request.Header["X-Client-Uuid"][0],
		ReceiveReqTime: start.String(),
		StartTime:      start_query.String(),
		EndTime:        fmt.Sprintf("%v", elapsed_query),
		ElapsedTime:    fmt.Sprintf("%v", elapsed),
	})

	c.IndentedJSON(http.StatusOK, nil)
}

func init() {
	// Disable log prefixes such as the default timestamp.
	// Prefix text prevents the message from being parsed as JSON.
	// A timestamp is added when shipping logs to Cloud Logging.
	log.SetFlags(0)
}

func main() {
	router := gin.Default()
	router.GET("/startTest", startTest)
	router.Run(":8080")
}
