package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/cloudspannerecosystem/spanner-cli/queryplan"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/genproto/googleapis/spanner/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

var (
	withStatsColumnNamesMap = map[bool][]string{
		false: {"ID", "Query_Execution_Plan (EXPERIMENTAL)"},
		true:  {"ID", "Query_Execution_Plan", "Rows_Returned", "Executions", "Total_Latency"},
	}
)

func _main() error {
	mode := flag.String("query-mode", "", "PROFILE or PLAN")
	flag.Parse()

	var withStats bool
	switch *mode {
	case "", "PROFILE":
		withStats = true
	case "PLAN":
		withStats = false
	default:
		flag.Usage()
		os.Exit(1)
	}

	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var qp spanner.QueryPlan
	err = protojson.Unmarshal(b, &qp)
	if err != nil {
		return err
	}

	rows, predicates, err := queryplan.ProcessPlanImpl(&qp, withStats)
	if err != nil {
		return err
	}

	printResult(os.Stdout, withStatsColumnNamesMap[withStats], rows, predicates)
	return nil
}

func printResult(out io.Writer, columns []string, rows [][]string, predicates []string) {
	table := tablewriter.NewWriter(out)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	for _, row := range rows {
		table.Append(row)
	}
	table.SetHeader(columns)
	if len(rows) > 0 {
		table.Render()
	}

	if len(predicates) > 0 {
		fmt.Fprintln(out, "Predicates(identified by ID):")
		for _, s := range predicates {
			fmt.Fprintf(out, " %s\n", s)
		}
		fmt.Fprintln(out)
	}
}
