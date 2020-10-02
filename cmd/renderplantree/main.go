package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cloudspannerecosystem/spanner-cli/internal/queryplan"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/genproto/googleapis/spanner/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

type tableRenderDef struct {
	ColumnsMapFunc func(queryplan.Row) []string
	ColumnNames    []string
}

var (
	withStatsToRenderDefMap = map[bool]tableRenderDef{
		false: {
			ColumnNames: []string{"ID", "Query_Execution_Plan (EXPERIMENTAL)"},
			ColumnsMapFunc: func(row queryplan.Row) []string {
				return []string{row.FormattedID, row.Text}
			},
		},
		true: {
			ColumnNames: []string{"ID", "Query_Execution_Plan", "Rows_Returned", "Executions", "Total_Latency"},
			ColumnsMapFunc: func(row queryplan.Row) []string {
				return []string{row.FormattedID, row.Text, row.RowsTotal, row.Execution, row.LatencyTotal}
			},
		},
	}
)

func _main() error {
	mode := flag.String("mode", "", "PROFILE or PLAN(ignore case)")
	flag.Parse()

	var withStats bool
	switch strings.ToUpper(*mode) {
	case "", "PLAN":
		withStats = false
	case "PROFILE":
		withStats = true
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

	rows, predicates, err := queryplan.ProcessPlan(&qp)
	if err != nil {
		return err
	}
	printResult(os.Stdout, withStatsToRenderDefMap[withStats], rows, predicates)
	return nil
}

func printResult(out io.Writer, renderDef tableRenderDef, rows []queryplan.Row, predicates []string) {
	table := tablewriter.NewWriter(out)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	for _, row := range rows {
		table.Append(renderDef.ColumnsMapFunc(row))
	}
	table.SetHeader(renderDef.ColumnNames)
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
