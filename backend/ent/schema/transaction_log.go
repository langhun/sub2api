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

// TransactionLog 定义不可变资金流水。
type TransactionLog struct {
	ent.Schema
}

func (TransactionLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "transactions_log"},
	}
}

func (TransactionLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("tx_id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Int64("user_id"),
		field.Int64("account_id"),
		field.String("tx_type").
			MaxLen(32).
			Immutable(),
		field.Other("amount", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Immutable(),
		field.Other("balance_before", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Immutable(),
		field.Other("balance_after", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Immutable(),
		field.Other("frozen_before", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Immutable(),
		field.Other("frozen_after", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Immutable(),
		field.Other("credit_limit_snapshot", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero).
			Immutable(),
		field.Other("debt_snapshot", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero).
			Immutable(),
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default("").
			Immutable(),
		field.String("reference_type").
			MaxLen(64).
			Optional().
			Nillable().
			Immutable(),
		field.String("reference_id").
			MaxLen(128).
			Optional().
			Nillable().
			Immutable(),
		field.String("request_id").
			MaxLen(128).
			Optional().
			Nillable().
			Immutable(),
		field.String("idempotency_scope").
			MaxLen(128).
			Immutable(),
		field.String("idempotency_key_hash").
			MaxLen(64).
			Immutable(),
		field.JSON("metadata", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Immutable(),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (TransactionLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("transaction_logs").
			Field("user_id").
			Unique().
			Required(),
		edge.From("account", UserBankAccount.Type).
			Ref("transactions").
			Field("account_id").
			Unique().
			Required(),
	}
}

func (TransactionLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tx_id").
			Unique(),
		index.Fields("idempotency_scope", "idempotency_key_hash").
			Unique(),
		index.Fields("user_id"),
		index.Fields("account_id"),
		index.Fields("tx_type"),
		index.Fields("created_at"),
		index.Fields("user_id", "created_at"),
		index.Fields("reference_type", "reference_id"),
	}
}
