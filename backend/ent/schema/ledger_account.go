package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LedgerAccount 定义 Core Banking 的总账账户科目。
type LedgerAccount struct {
	ent.Schema
}

func (LedgerAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ledger_accounts"},
	}
}

func (LedgerAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (LedgerAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_code").
			MaxLen(128),
		field.String("account_name").
			MaxLen(128),
		field.String("account_type").
			MaxLen(16),
		field.String("normal_balance").
			MaxLen(8),
		field.String("owner_type").
			MaxLen(16).
			Default("PLATFORM"),
		field.Int64("owner_user_id").
			Optional().
			Nillable(),
		field.Int64("user_bank_account_id").
			Optional().
			Nillable(),
		field.String("currency").
			MaxLen(16).
			Default("USD"),
		field.String("status").
			MaxLen(16).
			Default("ACTIVE"),
		field.JSON("metadata", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
	}
}

func (LedgerAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner_user", User.Type).
			Ref("ledger_accounts").
			Field("owner_user_id").
			Unique(),
		edge.From("bank_account", UserBankAccount.Type).
			Ref("ledger_accounts").
			Field("user_bank_account_id").
			Unique(),
		edge.To("entries", LedgerEntry.Type),
	}
}

func (LedgerAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_code").
			Unique(),
		index.Fields("owner_user_id"),
		index.Fields("user_bank_account_id"),
		index.Fields("account_type"),
		index.Fields("status"),
	}
}
