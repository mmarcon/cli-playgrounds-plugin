use("{{.DatabaseName}}");
db.{{.CollectionName}}
.aggregate(
{{.AggregationPipeline}}
)
.toArray();