package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	"cloud.google.com/go/spanner"
)

func queryWithByMutation(w io.Writer, client *spanner.Client, batchsize_p int) error {
	ctx := context.Background()

	mutations := make([]*spanner.Mutation, 0)

	columns := []string{
		"policyAcceptance", "registerAdSource", "registerFrom", "registerIp", "secret",
		"tag", "tmpPasswordHashed", "username", "uuid", "registerTime",
	}

	for i := 0; i < batchsize_p; i++ {
		registerTime := time.Now().UTC()
		uuidVal := uuid.NewString()
		policyAcceptance := int64(i % 6)
		registerFrom := int64(i % 6)

		mutation := spanner.InsertOrUpdate("User_partial", columns, []interface{}{
			policyAcceptance, "test", registerFrom, fmt.Sprintf("%d.%d.%d.%d", (i)%256, (i+i)%256, (i*i+1)%256, (i+5)%256),
			"b3230bb01425a0639e24793b6ab86131", "oasis-p-testing", "password", fmt.Sprintf("user_%s@example.com", uuidVal),
			uuidVal, registerTime,
		})

		mutations = append(mutations, mutation)
	}

	_, err := client.Apply(ctx, mutations)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
	}

	return err
}

func main() {
	ctx := context.Background()
	project := os.Getenv("PROJECT")
	instance := os.Getenv("INSTANCE")
	database := os.Getenv("DATABASE")
	// This database must exist.
	if project == "" {
		project = "williamlab"
		instance = "go-spanner-test-instance"
		database = "go-spanner-db"
	}
	databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		log.Fatalf("Failed to create client %v", err)
	}

	defer client.Close()

	args := os.Args
	batchsize, _ := strconv.Atoi(args[1])

	queryWithByMutation(os.Stdout, client, batchsize)
	log.Println("Insert Completed")
}
