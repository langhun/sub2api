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

type BalanceRedPacketClaim struct {
	ent.Schema
}

func (BalanceRedPacketClaim) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "balance_redpacket_claims"},
	}
}

func (BalanceRedPacketClaim) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("redpacket_id"),
		field.Int64("user_id"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int64("transfer_id").
			Optional().
			Nillable(),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (BalanceRedPacketClaim) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("redpacket", BalanceRedPacket.Type).
			Ref("claims").
			Field("redpacket_id").
			Unique().
			Required(),
	}
}

func (BalanceRedPacketClaim) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("redpacket_id", "user_id").Unique(),
		index.Fields("user_id"),
	}
}
