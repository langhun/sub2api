package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type CheckinPrizeItem struct {
	ent.Schema
}

func (CheckinPrizeItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "checkin_prize_items"},
	}
}

func (CheckinPrizeItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			SchemaType(map[string]string{dialect.Postgres: "varchar(100)"}),
		field.String("rarity").
			Default("common").
			SchemaType(map[string]string{dialect.Postgres: "varchar(20)"}),
		field.String("reward_type").
			Default("balance").
			SchemaType(map[string]string{dialect.Postgres: "varchar(30)"}),
		field.Float("reward_value").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("reward_value_max").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int64("subscription_id").
			Optional().
			Nillable(),
		field.Int("subscription_days").
			Default(0),
		field.Int("weight").
			Default(100),
		field.Bool("is_enabled").
			Default(true),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("deleted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CheckinPrizeItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_enabled"),
	}
}
