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

type BalanceRedPacket struct {
	ent.Schema
}

func (BalanceRedPacket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "balance_redpackets"},
	}
}

func (BalanceRedPacket) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("sender_id"),
		field.Float("total_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int("total_count"),
		field.Float("remaining_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int("remaining_count"),
		field.String("redpacket_type").
			MaxLen(20).
			Default("equal"),
		field.Float("fee").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("fee_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,6)"}),
		field.String("code").
			MaxLen(32).
			Unique(),
		field.String("status").
			MaxLen(20).
			Default("active"),
		field.Text("memo").
			Optional().
			Nillable(),
		field.Time("expire_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (BalanceRedPacket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sender", User.Type).
			Ref("redpackets").
			Field("sender_id").
			Unique().
			Required(),
		edge.To("claims", BalanceRedPacketClaim.Type),
	}
}

func (BalanceRedPacket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sender_id"),
		index.Fields("code").Unique(),
		index.Fields("status"),
		index.Fields("expire_at"),
	}
}
