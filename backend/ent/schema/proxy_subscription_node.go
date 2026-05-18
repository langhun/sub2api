package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxySubscriptionNode stores parsed proxy nodes from subscription sources.
type ProxySubscriptionNode struct {
	ent.Schema
}

func (ProxySubscriptionNode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "proxy_subscription_nodes"},
	}
}

func (ProxySubscriptionNode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (ProxySubscriptionNode) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("source_id"),
		field.String("node_key").
			MaxLen(256).
			NotEmpty(),
		field.String("display_name").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("node_type").
			MaxLen(32).
			NotEmpty(),
		field.String("server").
			MaxLen(255).
			NotEmpty(),
		field.Int("port"),
		field.JSON("config_json", map[string]any{}).
			Optional(),
		field.String("landing_status").
			MaxLen(32).
			Default("pending"),
		field.Text("last_error").
			Optional().
			Nillable(),
		field.Time("last_seen_at"),
	}
}

func (ProxySubscriptionNode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source", ProxySubscriptionSource.Type).
			Ref("nodes").
			Field("source_id").
			Unique().
			Required(),
	}
}

func (ProxySubscriptionNode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_id"),
		index.Fields("landing_status"),
		index.Fields("deleted_at"),
		index.Fields("source_id", "node_key").Unique(),
	}
}
