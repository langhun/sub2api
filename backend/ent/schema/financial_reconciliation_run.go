package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type FinancialReconciliationRun struct {
	ent.Schema
}

func (FinancialReconciliationRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "financial_reconciliation_runs"},
	}
}

func (FinancialReconciliationRun) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("run_id", uuid.UUID{}).
			Default(uuid.New),
		field.Time("run_date").
			SchemaType(map[string]string{dialect.Postgres: "date"}),
		field.String("status").
			MaxLen(16).
			Default("PENDING"),
		field.Int64("checked_transactions").
			Default(0),
		field.Int64("checked_ledger_entries").
			Default(0),
		field.Int64("mismatch_count").
			Default(0),
		field.JSON("summary", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("started_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("finished_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (FinancialReconciliationRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("issues", FinancialReconciliationIssue.Type),
	}
}

func (FinancialReconciliationRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id").Unique(),
		index.Fields("run_date"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
