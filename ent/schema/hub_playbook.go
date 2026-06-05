package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HubPlaybook represents a knowledge entry (decision, pattern, gotcha) shared
// across businesses. Cross-business when business_id IS NULL.
//
// Full-text search uses a tsvector column `tsv` populated by a trigger
// (see migration 0003). GIN index on tsv enables fast queries.
type HubPlaybook struct {
	ent.Schema
}

func (HubPlaybook) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "hub_playbooks",
			Schema: "hub",
		},
	}
}

func (HubPlaybook) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		SoftDeleteMixin{},
	}
}

func (HubPlaybook) Fields() []ent.Field {
	return []ent.Field{
		field.String("category").
			MaxLen(32).
			NotEmpty().
			Comment("decisions / patterns / gotchas"),
		field.String("title").
			MaxLen(256).
			NotEmpty(),
		field.Text("content").
			NotEmpty().
			Comment("Markdown 内容"),
		field.Strings("tags").
			Optional().
			Comment("标签数组"),
		field.String("tsv").
			MaxLen(1024).
			Optional().
			Comment("PostgreSQL tsvector，由 trigger 维护"),
		field.String("created_by_worker_id").
			MaxLen(128).
			Optional().
			Comment("上传者 worker"),
	}
}

func (HubPlaybook) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("playbooks").
			Unique(),  // nullable: 跨业务 playbook
	}
}

func (HubPlaybook) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("category"),
		// GIN(tsv) 由 migration 0003 创建（ent 不直接支持）
	}
}
