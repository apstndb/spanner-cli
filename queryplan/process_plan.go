package queryplan

import (
	"fmt"
	"strings"

	pb "google.golang.org/genproto/googleapis/spanner/v1"
)

func ProcessPlanImpl(plan *pb.QueryPlan, withStats bool) (rows [][]string, predicates []string, err error) {
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
		if withStats {
			rows = append(rows, []string{formattedID, row.Text, row.RowsTotal, row.Execution, row.LatencyTotal})
		} else {
			rows = append(rows, []string{formattedID, row.Text})
		}
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
