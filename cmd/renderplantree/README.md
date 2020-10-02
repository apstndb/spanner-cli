## How to use

```
$ go run ./ --mode=PLAN < ../../testdata/plans/scalar_subqueries.input.json
+-----+-----------------------------------------------------------------------+
| ID  | Query_Execution_Plan (EXPERIMENTAL)                                   |
+-----+-----------------------------------------------------------------------+
|   0 | Distributed Union                                                     |
|   1 | +- Local Distributed Union                                            |
|   2 |    +- Serialize Result                                                |
|   3 |       +- Index Scan (Full scan: true, Index: SingersByFirstLastName)  |
|  10 |       +- [Scalar] Scalar Subquery                                     |
|  11 |          +- Global Stream Aggregate (scalar_aggregate: true)          |
|  12 |             +- Distributed Union                                      |
|  13 |                +- Local Stream Aggregate (scalar_aggregate: true)     |
|  14 |                   +- Local Distributed Union                          |
| *15 |                      +- FilterScan                                    |
|  16 |                         +- Table Scan (Full scan: true, Table: Songs) |
+-----+-----------------------------------------------------------------------+
Predicates(identified by ID):
 15: Residual Condition: ($Duration > 300)
```

```
$ gcloud spanner databases execute-sql --project=${PROJECT_ID} --instance=${INSTANCE_ID} ${DATABASE_ID} \
  --sql 'SELECT SingerId FROM Singers WHERE FirstName LIKE "A%z"' --query-mode=PROFILE --format=json \
  | jq .stats.queryPlan | go run ./ --mode=PROFILE
+----+-------------------------------------------------------------------------+---------------+------------+---------------+
| ID | Query_Execution_Plan                                                    | Rows_Returned | Executions | Total_Latency |
+----+-------------------------------------------------------------------------+---------------+------------+---------------+
| *0 | Distributed Union                                                       | 0             | 1          | 4.08 msecs    |
|  1 | +- Local Distributed Union                                              | 0             | 1          | 3.96 msecs    |
|  2 |    +- Serialize Result                                                  | 0             | 1          | 3.95 msecs    |
| *3 |       +- FilterScan                                                     |               |            |               |
|  4 |          +- Index Scan (Full scan: true, Index: SingersByFirstLastName) | 0             | 1          | 3.94 msecs    |
+----+-------------------------------------------------------------------------+---------------+------------+---------------+
Predicates(identified by ID):
 0: Split Range: ($FirstName LIKE 'A%z')
 3: Residual Condition: ($FirstName LIKE 'A%z')
```
