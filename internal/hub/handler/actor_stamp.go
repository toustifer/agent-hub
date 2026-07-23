package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ActorStamp is the attribution written onto hub_events / audit rows.
type ActorStamp struct {
	UserID      int64
	Email       string
	Role        string // leader|worker|reviewer|system
	WorkerID    string
	LegacyActor string // hub_events.actor (legacy column)
}

func normalizeActorRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "leader", "worker", "reviewer", "system":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return "worker"
	}
}

// stampActorFromGin prefers JWT identity; never trusts client user_id.
func stampActorFromGin(c *gin.Context, clientRole, clientWorkerID, clientActor string) ActorStamp {
	st := ActorStamp{
		Role:     normalizeActorRole(clientRole),
		WorkerID: strings.TrimSpace(clientWorkerID),
	}
	if v, ok := c.Get("user_id"); ok && v != nil {
		switch id := v.(type) {
		case int64:
			st.UserID = id
		case float64:
			st.UserID = int64(id)
		case int:
			st.UserID = int64(id)
		}
	}
	if v, ok := c.Get("email"); ok {
		if s, ok := v.(string); ok {
			st.Email = strings.TrimSpace(s)
		}
	}
	switch {
	case st.UserID > 0 && st.Email != "":
		st.LegacyActor = st.Email
	case st.Email != "":
		st.LegacyActor = st.Email
	case strings.TrimSpace(clientActor) != "":
		st.LegacyActor = strings.TrimSpace(clientActor)
		if st.UserID == 0 && st.Role == "worker" && !strings.Contains(st.LegacyActor, "@") {
			// machine-ish actor string without JWT user
			if st.WorkerID == "" {
				st.WorkerID = st.LegacyActor
			}
		}
	case st.WorkerID != "":
		st.LegacyActor = st.WorkerID
		if st.UserID == 0 {
			st.Role = "system"
		}
	default:
		st.LegacyActor = "unknown"
		if st.UserID == 0 {
			st.Role = "system"
		}
	}
	return st
}

func userIDPtr(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	v := id
	return &v
}
