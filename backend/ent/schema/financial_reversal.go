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

type FinancialReversal struct {
	ent.Schema
}

func (FinancialReversal) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "financial_reversals"},
	}
}

func (FinancialReversal) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("reversal_id", uuid.UUID{}).
			Default(uuid.New),
		field.UUID("original_tx_id", uuid.UUID{}),
		field.UUID("reversal_tx_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Int64("requested_by_user_id").
			Optional().
			Nillable(),
		field.Int64("approved_by_user_id").
			Optional().
			Nillable(),
		field.String("request_id").
			MaxLen(128).
			Optional().
			Nillable(),
		field.String("reason").
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("status").
			MaxLen(16).
			Default("PENDING"),
		field.JSON("metadata", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("applied_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (FinancialReversal) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("requested_by_user", User.Type).
			Ref("requested_financial_reversals").
			Field("requested_by_user_id").
			Unique(),
		edge.From("approved_by_user", User.Type).
			Ref("approved_financial_reversals").
			Field("approved_by_user_id").
			Unique(),
	}
}

func (FinancialReversal) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("reversal_id").Unique(),
		index.Fields("original_tx_id").Unique(),
		index.Fields("reversal_tx_id"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
