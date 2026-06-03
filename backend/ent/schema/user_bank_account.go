package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/shopspring/decimal"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserBankAccount 定义用户虚拟银行账户。
type UserBankAccount struct {
	ent.Schema
}

func (UserBankAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users_bank_account"},
	}
}

func (UserBankAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (UserBankAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Other("balance", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Other("frozen_amount", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Other("credit_limit", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Other("total_debt", decimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric(38,18)"}).
			Default(decimal.Zero),
		field.Int64("version").
			Default(1),
		field.String("status").
			MaxLen(20).
			Default("ACTIVE"),
	}
}

func (UserBankAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("bank_account").
			Field("user_id").
			Unique().
			Required(),
		edge.To("transactions", TransactionLog.Type),
	}
}

func (UserBankAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").
			Unique(),
		index.Fields("status"),
	}
}
