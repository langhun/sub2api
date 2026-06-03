package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
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

// LoanContract 定义平台贷款和用户放贷合约。
type LoanContract struct {
	ent.Schema
}

func (LoanContract) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "loans_contract"},
	}
}

func (LoanContract) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (LoanContract) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("loan_id", uuid.UUID{}).
			Default(uuid.New),
		field.Int64("borrower_id"),
		field.String("lender_type").
			MaxLen(16),
		field.Int64("lender_id").
			Optional().
			Nillable(),
		field.Other("principal", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}),
		field.Other("interest_rate", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(18,12)"}),
		field.Other("accrued_interest", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Other("repaid_principal", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Other("repaid_interest", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.String("status").
			MaxLen(32).
			Default("ACTIVE"),
		field.Time("due_date").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (LoanContract) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("borrower", User.Type).
			Ref("borrowed_loan_contracts").
			Field("borrower_id").
			Unique().
			Required(),
		edge.From("lender", User.Type).
			Ref("funded_loan_contracts").
			Field("lender_id").
			Unique(),
	}
}

func (LoanContract) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("loan_id").
			Unique(),
		index.Fields("borrower_id"),
		index.Fields("lender_type", "lender_id"),
		index.Fields("status"),
		index.Fields("due_date"),
		index.Fields("status", "due_date"),
	}
}
