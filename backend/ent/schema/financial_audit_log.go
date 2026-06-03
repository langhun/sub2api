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

type FinancialAuditLog struct {
	ent.Schema
}

func (FinancialAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "financial_audit_logs"},
	}
}

func (FinancialAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("audit_id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("action").
			MaxLen(64).
			Immutable(),
		field.String("actor_type").
			MaxLen(16).
			Default("SYSTEM").
			Immutable(),
		field.Int64("actor_user_id").
			Optional().
			Nillable(),
		field.String("request_id").
			MaxLen(128).
			Optional().
			Nillable().
			Immutable(),
		field.UUID("tx_id", uuid.UUID{}).
			Optional().
			Nillable().
			Immutable(),
		field.UUID("reversal_id", uuid.UUID{}).
			Optional().
			Nillable().
			Immutable(),
		field.String("target_type").
			MaxLen(64).
			Immutable(),
		field.String("target_id").
			MaxLen(128).
			Immutable(),
		field.JSON("metadata", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Immutable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (FinancialAuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("actor_user", User.Type).
			Ref("financial_audit_logs").
			Field("actor_user_id").
			Unique(),
	}
}

func (FinancialAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("audit_id").Unique(),
		index.Fields("action"),
		index.Fields("tx_id"),
		index.Fields("target_type", "target_id"),
		index.Fields("created_at"),
	}
}
