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

// LedgerEntry 定义 Core Banking 的不可变借贷分录。
type LedgerEntry struct {
	ent.Schema
}

func (LedgerEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ledger_entries"},
	}
}

func (LedgerEntry) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("entry_id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Int64("transaction_log_id").
			Comment("关联的用户可读交易流水 ID"),
		field.UUID("tx_id", uuid.UUID{}).
			Immutable(),
		field.Int64("ledger_account_id").
			Comment("关联的总账账户 ID"),
		field.Int64("user_id").
			Optional().
			Nillable().
			Comment("用户相关分录的用户 ID，平台分录可为空"),
		field.String("entry_side").
			MaxLen(8).
			Immutable(),
		field.Other("amount", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Immutable(),
		field.String("business_module").
			MaxLen(32).
			Immutable(),
		field.String("tx_type").
			MaxLen(32).
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
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default("").
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

func (LedgerEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("transaction", TransactionLog.Type).
			Ref("ledger_entries").
			Field("transaction_log_id").
			Unique().
			Required(),
		edge.From("ledger_account", LedgerAccount.Type).
			Ref("entries").
			Field("ledger_account_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("ledger_entries").
			Field("user_id").
			Unique(),
	}
}

func (LedgerEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("entry_id").
			Unique(),
		index.Fields("transaction_log_id"),
		index.Fields("tx_id"),
		index.Fields("ledger_account_id"),
		index.Fields("user_id"),
		index.Fields("business_module", "tx_type"),
		index.Fields("created_at"),
		index.Fields("reference_type", "reference_id"),
	}
}
