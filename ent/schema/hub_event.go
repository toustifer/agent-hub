package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HubEvent struct {
	ent.Schema
}

func (HubEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hub_events", Schema: "hub"},
	}
}

func (HubEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (HubEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("business_id"),
		field.String("actor").MaxLen(128).NotEmpty(),
		field.String("event_type").MaxLen(64).NotEmpty(),
		field.JSON("payload", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
	}
}

func (HubEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("events").Field("business_id").Unique().Required(),
	}
}

func (HubEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type"),
		index.Fields("created_at"),
		index.Fields("business_id", "created_at"),
	}
}
