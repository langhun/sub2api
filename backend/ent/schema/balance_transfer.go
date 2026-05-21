package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type BalanceTransfer struct {
	ent.Schema
}

func (BalanceTransfer) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "balance_transfers"},
	}
}

func (BalanceTransfer) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("sender_id"),
		field.Int64("receiver_id"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("fee").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("fee_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,6)"}),
		field.Float("gross_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("transfer_type").
			MaxLen(20).
			Default("direct"),
		field.String("status").
			MaxLen(20).
			Default("completed"),
		field.Text("memo").
			Optional().
			Nillable(),
		field.Int64("redpacket_id").
			Optional().
			Nillable(),
		field.Time("frozen_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int64("frozen_by").
			Optional().
			Nillable(),
		field.Text("revoke_reason").
			Optional().
			Nillable(),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (BalanceTransfer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sender", User.Type).
			Ref("sent_transfers").
			Field("sender_id").
			Unique().
			Required(),
		edge.From("receiver", User.Type).
			Ref("received_transfers").
			Field("receiver_id").
			Unique().
			Required(),
	}
}

func (BalanceTransfer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sender_id"),
		index.Fields("receiver_id"),
		index.Fields("status"),
		index.Fields("transfer_type"),
		index.Fields("created_at"),
		index.Fields("sender_id", "created_at"),
		index.Fields("receiver_id", "created_at"),
	}
}
