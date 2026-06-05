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

// HubEvent is an append-only audit log entry.
//
// Events are NEVER deleted; the data is for debugging, auditing, and SSE streaming.
type HubEvent struct {
	ent.Schema
}

func (HubEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "hub_events",
			Schema: "hub",
		},
	}
}

func (HubEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (HubEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("actor").
			MaxLen(128).
			NotEmpty().
			Comment("谁触发（worker_id 或 user_email）"),
		field.String("event_type").
			MaxLen(64).
			NotEmpty().
			Comment("lock.acquired / lock.released / heartbeat.missed / business.registered ..."),
		field.JSON("payload", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (HubEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("events").
			Unique().
			Required(),
	}
}

func (HubEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type"),
		index.Fields("created_at"),
	}
}
