package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// SoftDeleteMixin adds a nullable deleted_at timestamp field.
//
// Note: hub's simplified version. The mixin only adds the field; callers must
// explicitly use `Where(hub.DeletedAtIsNil())` in queries, and manually set
// `deleted_at = now()` for soft-delete. We chose this over an Interceptor-based
// approach to avoid reflect/mutation complexity (see sub2api's full version
// for reference if needed).
type SoftDeleteMixin struct {
	mixin.Schema
}

func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}
