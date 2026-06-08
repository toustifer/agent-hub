package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HubPlaybook struct {
	ent.Schema
}

func (HubPlaybook) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hub_playbooks", Schema: "hub"},
	}
}

func (HubPlaybook) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}, SoftDeleteMixin{}}
}

func (HubPlaybook) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("business_id").Optional().Nillable(),
		field.String("category").MaxLen(32).NotEmpty(),
		field.String("title").MaxLen(256).NotEmpty(),
		field.Text("content").NotEmpty(),
		field.Strings("tags").Optional(),
		field.String("tsv").MaxLen(1024).Optional(),
		field.String("created_by_worker_id").MaxLen(128).Optional(),
	}
}

func (HubPlaybook) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business", HubBusiness.Type).
			Ref("playbooks").Field("business_id").Unique(),
	}
}

func (HubPlaybook) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("category"),
	}
}
