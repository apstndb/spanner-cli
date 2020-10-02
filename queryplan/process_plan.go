package queryplan

import (
	"fmt"
	"strings"

	pb "google.golang.org/genproto/googleapis/spanner/v1"
)

type Row struct {
	FormattedID  string
	Text         string
	RowsTotal    string
	Execution    string
	LatencyTotal string
}

func ProcessPlan(plan *pb.QueryPlan) (rows []Row, predicates []string, err error) {
	planNodes := plan.GetPlanNodes()
	maxWidthOfNodeID := len(fmt.Sprint(getMaxVisibleNodeID(plan)))
	widthOfNodeIDWithIndicator := maxWidthOfNodeID + 1

	tree := BuildQueryPlanTree(plan, 0)

	treeRows, err := tree.RenderTreeWithStats(planNodes)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range treeRows {
		var formattedID string
		if len(row.Predicates) > 0 {
			formattedID = fmt.Sprintf("%*s", widthOfNodeIDWithIndicator, "*"+fmt.Sprint(row.ID))
		} else {
			formattedID = fmt.Sprintf("%*d", widthOfNodeIDWithIndicator, row.ID)
		}
		rows = append(rows, Row{
			FormattedID:  formattedID,
			Text:         row.Text,
			RowsTotal:    row.RowsTotal,
			Execution:    row.Execution,
			LatencyTotal: row.LatencyTotal,
		})
		for i, predicate := range row.Predicates {
			var prefix string
			if i == 0 {
				prefix = fmt.Sprintf("%*d:", maxWidthOfNodeID, row.ID)
			} else {
				prefix = strings.Repeat(" ", maxWidthOfNodeID+1)
			}
			predicates = append(predicates, fmt.Sprintf("%s %s", prefix, predicate))
		}
	}
	return rows, predicates, nil
}
