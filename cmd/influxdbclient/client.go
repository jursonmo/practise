package main

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var server = "http://192.168.64.5:8086"
var token = "FNMW4EKu4rOfm1AH8JzrYcswmf1ETYoWA_RhuUPVH2GffUe4HTL7dlagJeO36_FCiwvjrGkuKjOp2RGEdDznBA=="
var org = "org1"
var bucket = "bucket1"

func main() {
	//influxdb2.NewClientWithOptions()
	// Create a new client using an InfluxDB server base URL and an authentication token
	client := influxdb2.NewClient(server, token)
	// Use blocking write client for writes to desired bucket
	writeAPI := client.WriteAPIBlocking(org, bucket)
	// Create point using full params constructor
	p := influxdb2.NewPoint("stat",
		map[string]string{"unit": "temperature"},
		map[string]interface{}{"avg": 11.0, "max": 22.0},
		time.Now())
	// write point immediately
	err := writeAPI.WritePoint(context.Background(), p)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create point using fluent style
	p = influxdb2.NewPointWithMeasurement("stat").
		AddTag("unit", "temperature").
		AddField("avg", 33.2).
		AddField("max", 44.0).
		SetTime(time.Now())
	err = writeAPI.WritePoint(context.Background(), p)
	if err != nil {
		fmt.Println(err)
		return
	}
	/*
		// Or write directly line protocol
		line := fmt.Sprintf("stat,unit=temperature avg=%f,max=%f", 55.5, 66.0)
		err = writeAPI.WriteRecord(context.Background(), line)
		if err != nil {
			fmt.Println(err)
			return
		}
		// Get query client
		queryAPI := client.QueryAPI("org1")
		// Get parser flux query result
		result, err := queryAPI.Query(context.Background(), `from(bucket:"bucket1")|> range(start: -1h) |> filter(fn: (r) => r._measurement == "stat")`)
		if err == nil {
			// Use Next() to iterate over query result lines
			for result.Next() {
				// Observe when there is new grouping key producing new table
				if result.TableChanged() {
					fmt.Printf("table: %s\n", result.TableMetadata().String())
				}
				// read result
				fmt.Printf("row: %s\n", result.Record().String())
			}
			if result.Err() != nil {
				fmt.Printf("Query error: %s\n", result.Err().Error())
			}
		}
		if err != nil {
			fmt.Println(err)
		}
	*/
	// Ensures background processes finishes
	client.Close()
}
