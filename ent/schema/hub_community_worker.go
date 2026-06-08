package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HubCommunityWorker struct {
	ent.Schema
}

func (HubCommunityWorker) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hub_community_workers", Schema: "hub"},
	}
}

func (HubCommunityWorker) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (HubCommunityWorker) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("publisher_user_id"),
		field.Text("title").NotEmpty(),
		field.Text("description").Optional(),
		field.String("domain").MaxLen(32).NotEmpty(),
		field.Text("scope").Optional(),
		field.JSON("handbook", map[string]interface{}{}).Optional(),
		field.JSON("playbooks", map[string]interface{}{}).Optional(),
		field.Strings("tags").Optional(),
		field.Int("install_count").Default(0),
		field.String("status").MaxLen(20).Default("published"),
	}
}

func (HubCommunityWorker) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("reviews", HubCommunityReview.Type),
	}
}

func (HubCommunityWorker) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("domain"),
		index.Fields("install_count"),
		index.Fields("publisher_user_id"),
	}
}
