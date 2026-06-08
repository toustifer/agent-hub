package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type HubLock struct {
	ent.Schema
}

func (HubLock) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hub_locks", Schema: "hub"},
	}
}

func (HubLock) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (HubLock) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("business_id"),
		field.String("resource_key").MaxLen(256).NotEmpty(),
		field.String("holder_token").MaxLen(64).NotEmpty(),
		field.String("holder_worker_id").MaxLen(128).NotEmpty(),
		field.Time("acquired_at"),
		field.Time("expires_at"),
		field.Time("heartbeat_at"),
		field.Time("released_at").Optional().Nillable(),
	}
}

func (HubLock) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("locks").Field("business_id").Unique().Required(),
	}
}
