package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HubWorker struct {
	ent.Schema
}

func (HubWorker) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hub_workers", Schema: "hub"},
	}
}

func (HubWorker) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (HubWorker) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("business_id"),
		field.String("worker_id").MaxLen(128).NotEmpty(),
		field.String("version").MaxLen(32).Optional(),
		field.Time("last_heartbeat_at").Optional().Nillable(),
		field.String("status").MaxLen(20).Default("offline"),
		field.String("host").MaxLen(128).Optional(),
		field.Int("pid").Optional().Nillable(),
		field.String("owner").MaxLen(64).Optional(),
		field.JSON("handbook", map[string]interface{}{}).Optional(),
	}
}

func (HubWorker) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("workers").Field("business_id").Unique().Required(),
	}
}

func (HubWorker) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("business_id", "worker_id").Unique(),
		index.Fields("status"),
		index.Fields("last_heartbeat_at"),
	}
}
