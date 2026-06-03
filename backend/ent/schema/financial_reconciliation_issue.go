package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type FinancialReconciliationIssue struct {
	ent.Schema
}

func (FinancialReconciliationIssue) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "financial_reconciliation_issues"},
	}
}

func (FinancialReconciliationIssue) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("issue_id", uuid.UUID{}).
			Default(uuid.New),
		field.Int64("reconciliation_id"),
		field.String("issue_type").
			MaxLen(32),
		field.UUID("tx_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Int64("ledger_account_id").
			Optional().
			Nillable(),
		field.Other("expected_amount", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Other("actual_amount", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.String("detail").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.JSON("metadata", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (FinancialReconciliationIssue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("reconciliation_run", FinancialReconciliationRun.Type).
			Ref("issues").
			Field("reconciliation_id").
			Unique().
			Required(),
		edge.From("ledger_account", LedgerAccount.Type).
			Ref("reconciliation_issues").
			Field("ledger_account_id").
			Unique(),
	}
}

func (FinancialReconciliationIssue) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("issue_id").Unique(),
		index.Fields("reconciliation_id"),
		index.Fields("issue_type"),
		index.Fields("tx_id"),
	}
}
