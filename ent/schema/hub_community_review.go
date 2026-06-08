package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type HubCommunityReview struct {
	ent.Schema
}

func (HubCommunityReview) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hub_community_reviews", Schema: "hub"},
	}
}

func (HubCommunityReview) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (HubCommunityReview) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("worker_id"),
		field.Int64("user_id"),
		field.Int("rating").Range(1, 5),
		field.Text("comment").Optional(),
	}
}

func (HubCommunityReview) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("community_worker", HubCommunityWorker.Type).
			Ref("reviews").
			Field("worker_id").
			Unique().
			Required(),
	}
}
