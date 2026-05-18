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

// ProxySubscriptionSource stores proxy subscription source definitions.
type ProxySubscriptionSource struct {
	ent.Schema
}

func (ProxySubscriptionSource) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "proxy_subscription_sources"},
	}
}

func (ProxySubscriptionSource) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (ProxySubscriptionSource) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("url").
			MaxLen(2048).
			NotEmpty(),
		field.String("source_format").
			MaxLen(32).
			Default("auto"),
		field.Bool("enabled").
			Default(true),
		field.Int("refresh_interval_hours").
			Default(6),
		field.Bool("auto_add_to_pool").
			Default(false),
		field.Time("last_refreshed_at").
			Optional().
			Nillable(),
		field.Time("last_success_at").
			Optional().
			Nillable(),
		field.Text("last_error").
			Optional().
			Nillable(),
		field.Int("last_node_count").
			Default(0),
		field.Int("last_materialized_proxy_count").
			Default(0),
	}
}

func (ProxySubscriptionSource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("nodes", ProxySubscriptionNode.Type),
	}
}

func (ProxySubscriptionSource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
		index.Fields("source_format"),
		index.Fields("deleted_at"),
	}
}
