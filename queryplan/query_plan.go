//
// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package queryplan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/xlab/treeprint"
	pb "google.golang.org/genproto/googleapis/spanner/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func init() {
	// Use only ascii characters
	treeprint.EdgeTypeLink = "|"
	treeprint.EdgeTypeMid = "+-"
	treeprint.EdgeTypeEnd = "+-"

	treeprint.IndentSize = 2
}

type link struct {
	Dest *node
	Type string
}

type node struct {
	PlanNode *pb.PlanNode
	Children []*link
}

type nodeWithStats struct {
	ID             int32            `json:"id"`
	ExecutionStats *structpb.Struct `json:"execution_stats"`
	DisplayName    string           `json:"display_name"`
	LinkType       string           `json:"link_type"`
}

type executionStatsValue struct {
	Unit  string `json:"unit"`
	Total string `json:"total"`
}

func (v executionStatsValue) String() string {
	if v.Unit == "" {
		return v.Total
	} else {
		return fmt.Sprintf("%s %s", v.Total, v.Unit)
	}
}

// nodeWithStatsTyped is proto-free typed representation of nodeWithStats
type nodeWithStatsTyped struct {
	ID             int32 `json:"id"`
	ExecutionStats struct {
		Rows             executionStatsValue `json:"rows"`
		Latency          executionStatsValue `json:"latency"`
		ExecutionSummary struct {
			NumExecutions string `json:"num_executions"`
		} `json:"execution_summary"`
	} `json:"execution_stats"`
	DisplayName string `json:"display_name"`
	LinkType    string `json:"link_type"`
}

func buildQueryPlanTree(plan *pb.QueryPlan, idx int32) *node {
	if len(plan.PlanNodes) == 0 {
		return &node{}
	}

	nodeMap := map[int32]*pb.PlanNode{}
	for _, node := range plan.PlanNodes {
		nodeMap[node.Index] = node
	}

	root := &node{
		PlanNode: plan.PlanNodes[idx],
		Children: make([]*link, 0),
	}
	if root.PlanNode.ChildLinks != nil {
		for i, childLink := range root.PlanNode.ChildLinks {
			idx := childLink.ChildIndex
			child := buildQueryPlanTree(plan, idx)
			childType := childLink.Type

			// Fill missing Input type into the first child of [Distributed] (Cross|Outer) Apply
			if childType == "" && strings.HasSuffix(root.PlanNode.DisplayName, "Apply") && i == 0 {
				childType = "Input"
			}
			root.Children = append(root.Children, &link{Type: childType, Dest: child})
		}
	}

	return root
}

type rowWithPredicates struct {
	ID           int32
	Text         string
	RowsTotal    string
	Execution    string
	LatencyTotal string
	Predicates   []string
}

func isPredicate(planNodes []*pb.PlanNode, childLink *pb.PlanNode_ChildLink) bool {
	// Known predicates are Condition(Filter/Hash Join) or Seek Condition/Residual Condition(FilterScan) or Split Range(Distributed Union).
	// Agg is a Function but not a predicate.
	child := planNodes[childLink.ChildIndex]
	if child.DisplayName != "Function" {
		return false
	}
	if strings.HasSuffix(childLink.GetType(), "Condition") || childLink.GetType() == "Split Range" {
		return true
	}
	return false
}

func (n *node) RenderTreeWithStats(planNodes []*pb.PlanNode) ([]rowWithPredicates, error) {
	tree := treeprint.New()
	renderTreeWithStats(tree, "", n)
	var result []rowWithPredicates
	for _, line := range strings.Split(tree.String(), "\n") {
		if line == "" {
			continue
		}

		split := strings.SplitN(line, "\t", 2)
		// Handle the case of the root node of treeprint
		if len(split) != 2 {
			return nil, fmt.Errorf("unexpected split error, tree line = %q", line)
		}
		branchText, protojsonText := split[0], split[1]

		var planNode nodeWithStatsTyped
		if err := json.Unmarshal([]byte(protojsonText), &planNode); err != nil {
			return nil, fmt.Errorf("unexpected JSON unmarshal error, tree line = %q", line)
		}

		var text string
		if planNode.LinkType != "" {
			text = fmt.Sprintf("[%s] %s", planNode.LinkType, planNode.DisplayName)
		} else {
			text = planNode.DisplayName
		}

		var predicates []string
		for _, cl := range planNodes[planNode.ID].GetChildLinks() {
			if !isPredicate(planNodes, cl) {
				continue
			}
			predicates = append(predicates, fmt.Sprintf("%s: %s", cl.GetType(), planNodes[cl.ChildIndex].GetShortRepresentation().GetDescription()))
		}

		result = append(result, rowWithPredicates{
			ID:           planNode.ID,
			Predicates:   predicates,
			Text:         branchText + text,
			RowsTotal:    planNode.ExecutionStats.Rows.Total,
			Execution:    planNode.ExecutionStats.ExecutionSummary.NumExecutions,
			LatencyTotal: planNode.ExecutionStats.Latency.String(),
		})
	}
	return result, nil
}

func (n *node) IsVisible() bool {
	operator := n.PlanNode.DisplayName
	if operator == "Function" || operator == "Reference" || operator == "Constant" || operator == "Array Constructor" || operator == "Parameter" {
		return false
	}

	return true
}

func (n *node) IsRoot() bool {
	return n.PlanNode.Index == 0
}

func (n *node) String() string {
	metadataFields := n.PlanNode.GetMetadata().GetFields()

	var operator string
	{
		var components []string
		for _, s := range []string{
			metadataFields["call_type"].GetStringValue(),
			metadataFields["iterator_type"].GetStringValue(),
			strings.TrimSuffix(metadataFields["scan_type"].GetStringValue(), "Scan"),
			n.PlanNode.GetDisplayName(),
		} {
			if s != "" {
				components = append(components, s)
			}
		}
		operator = strings.Join(components, " ")
	}

	var metadata string
	{
		fields := make([]string, 0)
		for k, v := range metadataFields {
			switch k {
			case "call_type", "iterator_type": // Skip because it is displayed in node title
				continue
			case "scan_target": // Skip because it is combined with scan_type
				continue
			case "subquery_cluster_node": // Skip because it is useless without displaying node id
				continue
			case "scan_type":
				fields = append(fields, fmt.Sprintf("%s: %s",
					strings.TrimSuffix(v.GetStringValue(), "Scan"),
					metadataFields["scan_target"].GetStringValue()))
			default:
				fields = append(fields, fmt.Sprintf("%s: %s", k, v.GetStringValue()))
			}
		}

		sort.Strings(fields)

		if len(fields) != 0 {
			metadata = fmt.Sprintf(`(%s)`, strings.Join(fields, ", "))
		}
	}

	if metadata == "" {
		return operator
	}
	return operator + " " + metadata
}

func renderTreeWithStats(tree treeprint.Tree, linkType string, node *node) {
	if !node.IsVisible() {
		return
	}

	b, _ := json.Marshal(
		nodeWithStats{
			ID:             node.PlanNode.Index,
			ExecutionStats: node.PlanNode.GetExecutionStats(),
			DisplayName:    node.String(),
			LinkType:       linkType,
		},
	)
	// Prefixed by tab to ease to split
	str := "\t" + string(b)

	if len(node.Children) > 0 {
		var branch treeprint.Tree
		if node.IsRoot() {
			tree.SetValue(str)
			branch = tree
		} else {
			branch = tree.AddBranch(str)
		}
		for i, child := range node.Children {
			// Serialize Result with scalar subqueries can have duplicate children
			// so process only the first child and children have Scalar type.
			if node.PlanNode.DisplayName == "Serialize Result" && !(i == 0 || child.Type == "Scalar") {
				continue
			}
			renderTreeWithStats(branch, child.Type, child.Dest)
		}
	} else {
		if node.IsRoot() {
			tree.SetValue(str)
		} else {
			tree.AddNode(str)
		}
	}
}

func getMaxVisibleNodeID(plan *pb.QueryPlan) int32 {
	var maxVisibleNodeID int32
	// We assume that plan_nodes[] is pre-sorted in ascending order.
	// See QueryPlan.plan_nodes[] in the document.
	// https://cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1?hl=en#google.spanner.v1.QueryPlan.FIELDS.repeated.google.spanner.v1.PlanNode.google.spanner.v1.QueryPlan.plan_nodes
	for _, planNode := range plan.GetPlanNodes() {
		if (&node{PlanNode: planNode}).IsVisible() {
			maxVisibleNodeID = planNode.Index
		}
	}
	return maxVisibleNodeID
}
