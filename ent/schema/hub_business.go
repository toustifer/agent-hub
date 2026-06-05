package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HubBusiness represents a registered business (e.g. siruoning, insight-tutor).
//
// It is the top-level entity: every Worker, Lock, Playbook, Event is scoped
// to a Business via business_id.
type HubBusiness struct {
	ent.Schema
}

func (HubBusiness) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "hub_businesses",
			Schema: "hub",
		},
	}
}

func (HubBusiness) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (HubBusiness) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			MaxLen(64).
			NotEmpty().
			Unique().
			Comment("业务代号，如 siruoning"),
		field.String("name").
			MaxLen(128).
			NotEmpty().
			Comment("业务名"),
		field.String("repo_url").
			MaxLen(512).
			Optional().
			Comment("仓库地址"),
		field.Int64("owner_user_id").
			Optional().
			Nillable().
			Comment("关联 sub2api users.id（只读引用）"),
		field.String("description").
			MaxLen(1024).
			Optional().
			Comment("业务说明"),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("active / suspended / archived"),
	}
}

func (HubBusiness) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
	}
}
