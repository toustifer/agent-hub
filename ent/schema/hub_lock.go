package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// HubLock represents a distributed lock on a resource.
//
// Concurrency safety comes from a PARTIAL UNIQUE INDEX on resource_key
// (where released_at IS NULL AND expires_at > now()), not from advisory locks.
// This is created in migration 0002, since ent cannot express partial indexes
// declaratively.
type HubLock struct {
	ent.Schema
}

func (HubLock) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "hub_locks",
			Schema: "hub",
		},
	}
}

func (HubLock) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (HubLock) Fields() []ent.Field {
	return []ent.Field{
		field.String("resource_key").
			MaxLen(256).
			NotEmpty().
			Comment("锁的资源，如 siruoning.medication.pages.Homepage"),
		field.String("holder_token").
			MaxLen(64).
			NotEmpty().
			Comment("持有者 token（每次 acquire 重新生成）"),
		field.String("holder_worker_id").
			MaxLen(128).
			NotEmpty().
			Comment("持有者 worker 名"),
		field.Time("acquired_at").
			NotEmpty().
			Comment("获取时间"),
		field.Time("expires_at").
			NotEmpty().
			Comment("过期时间"),
		field.Time("heartbeat_at").
			NotEmpty().
			Comment("最后续期时间"),
		field.Time("released_at").
			Optional().
			Nillable().
			Comment("主动释放时间（NULL = 还锁着）"),
	}
}

func (HubLock) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("locks").
			Unique().
			Required(),
	}
}
