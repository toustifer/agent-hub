package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HubWorker represents an AI worker (e.g. worker-medication) belonging to a Business.
//
// It sends heartbeat periodically. status is updated based on heartbeat freshness.
type HubWorker struct {
	ent.Schema
}

func (HubWorker) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "hub_workers",
			Schema: "hub",
		},
	}
}

func (HubWorker) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (HubWorker) Fields() []ent.Field {
	return []ent.Field{
		field.String("worker_id").
			MaxLen(128).
			NotEmpty().
			Comment("业务内的 worker 名，如 medication"),
		field.String("version").
			MaxLen(32).
			Optional().
			Comment("agent-company 版本"),
		field.Time("last_heartbeat_at").
			Optional().
			Nillable().
			Comment("最后心跳时间"),
		field.String("status").
			MaxLen(20).
			Default("offline").
			Comment("online / offline / dead"),
		field.String("host").
			MaxLen(128).
			Optional().
			Comment("Worker 所在主机名"),
		field.Int("pid").
			Optional().
			Nillable().
			Comment("Worker 进程 PID"),
	}
}

func (HubWorker) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("workers").
			Unique().
			Required(),
	}
}

func (HubWorker) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("business_id", "worker_id").
			Unique(),
		index.Fields("status"),
		index.Fields("last_heartbeat_at"),
	}
}
