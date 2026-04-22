package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/vectorcore/eag/internal/models"
)

func registerAlertHandlers(api huma.API, db *gorm.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "list-alerts",
		Method:      http.MethodGet,
		Path:        "/api/v1/alerts",
		Summary:     "List and search alerts",
		Tags:        []string{"Alerts"},
	}, func(ctx context.Context, input *ListAlertsInput) (*ListAlertsOutput, error) {
		return listAlerts(db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-alert-stats",
		Method:      http.MethodGet,
		Path:        "/api/v1/alerts/stats",
		Summary:     "Alert statistics",
		Tags:        []string{"Alerts"},
	}, func(ctx context.Context, input *struct{}) (*AlertStatsOutput, error) {
		return getAlertStats(db)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-alert",
		Method:      http.MethodGet,
		Path:        "/api/v1/alerts/{id}",
		Summary:     "Get alert by ID",
		Tags:        []string{"Alerts"},
	}, func(ctx context.Context, input *GetAlertInput) (*GetAlertOutput, error) {
		return getAlert(db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-alert",
		Method:      http.MethodDelete,
		Path:        "/api/v1/alerts/{id}",
		Summary:     "Soft-delete an alert",
		Tags:        []string{"Alerts"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *GetAlertInput) (*struct{}, error) {
		if err := db.Where("id = ?", input.ID).Delete(&models.Alert{}).Error; err != nil {
			return nil, huma.Error404NotFound("alert not found")
		}
		return nil, nil
	})
}

// --- Input/Output types ---

type ListAlertsInput struct {
	Q              string `query:"q"               doc:"Full-text search across headline, description, areaDesc, event"`
	Severity       string `query:"severity"        doc:"Comma-separated severity values"`
	Urgency        string `query:"urgency"         doc:"Comma-separated urgency values"`
	Certainty      string `query:"certainty"       doc:"Comma-separated certainty values"`
	Event          string `query:"event"           doc:"Partial match on event"`
	Area           string `query:"area"            doc:"Partial match on areaDesc"`
	FeedSource     string `query:"feed_source"     doc:"Exact feed source name"`
	Status         string `query:"status"          doc:"Comma-separated: Actual,Exercise,Test,Draft"`
	MsgType        string `query:"msg_type"        doc:"Comma-separated: Alert,Update,Cancel"`
	From           string `query:"from"            doc:"ISO 8601 — sent >= from"`
	To             string `query:"to"              doc:"ISO 8601 — sent <= to"`
	Forwarded      string `query:"forwarded"       doc:"Filter by forwarded state: true or false"`
	IncludeExpired bool   `query:"include_expired" doc:"Include soft-deleted records"`
	Page           int    `query:"page"            default:"1"`
	Limit          int    `query:"limit"           default:"50"  minimum:"1" maximum:"200"`
	Sort           string `query:"sort"            default:"sent"`
	Order          string `query:"order"           default:"desc"`
}

type ListAlertsOutput struct {
	Body struct {
		Total  int64          `json:"total"`
		Page   int            `json:"page"`
		Limit  int            `json:"limit"`
		Alerts []models.Alert `json:"alerts"`
	}
}

type GetAlertInput struct {
	ID string `path:"id"`
}

type GetAlertOutput struct {
	Body models.Alert
}

type AlertStatsOutput struct {
	Body struct {
		TotalActive              int64            `json:"total_active"`
		BySeverity               map[string]int64 `json:"by_severity"`
		ByStatus                 map[string]int64 `json:"by_status"`
		ByFeed                   map[string]int64 `json:"by_feed"`
		Forwarded                int64            `json:"forwarded"`
		PendingForward           int64            `json:"pending_forward"`
		DestinationsConfigured   bool             `json:"destinations_configured"`
	}
}

// --- Handlers ---

func listAlerts(db *gorm.DB, input *ListAlertsInput) (*ListAlertsOutput, error) {
	q := db.Model(&models.Alert{})

	if input.IncludeExpired {
		q = q.Unscoped()
	}

	if input.Q != "" {
		like := "%" + input.Q + "%"
		q = q.Where("headline LIKE ? OR description LIKE ? OR area_desc LIKE ? OR event LIKE ?",
			like, like, like, like)
	}
	if input.Severity != "" {
		q = q.Where("severity IN ?", splitCSV(input.Severity))
	}
	if input.Urgency != "" {
		q = q.Where("urgency IN ?", splitCSV(input.Urgency))
	}
	if input.Certainty != "" {
		q = q.Where("certainty IN ?", splitCSV(input.Certainty))
	}
	if input.Event != "" {
		q = q.Where("event LIKE ?", "%"+input.Event+"%")
	}
	if input.Area != "" {
		q = q.Where("area_desc LIKE ?", "%"+input.Area+"%")
	}
	if input.FeedSource != "" {
		q = q.Where("feed_source = ?", input.FeedSource)
	}
	if input.Status != "" {
		q = q.Where("status IN ?", splitCSV(input.Status))
	}
	if input.MsgType != "" {
		q = q.Where("msg_type IN ?", splitCSV(input.MsgType))
	}
	if input.From != "" {
		t, err := time.Parse(time.RFC3339, input.From)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid 'from' date: " + err.Error())
		}
		q = q.Where("sent >= ?", t)
	}
	if input.To != "" {
		t, err := time.Parse(time.RFC3339, input.To)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid 'to' date: " + err.Error())
		}
		q = q.Where("sent <= ?", t)
	}
	if input.Forwarded == "true" {
		q = q.Where("forwarded = ?", true)
	} else if input.Forwarded == "false" {
		q = q.Where("forwarded = ?", false)
	}

	var total int64
	q.Count(&total) //nolint:errcheck

	// Validate sort field to prevent SQL injection
	allowedSorts := map[string]bool{
		"sent": true, "expires": true, "severity": true, "urgency": true,
		"event": true, "area_desc": true, "feed_source": true, "created_at": true,
	}
	sort := input.Sort
	if !allowedSorts[sort] {
		sort = "sent"
	}
	order := strings.ToLower(input.Order)
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 {
		input.Limit = 50
	}
	offset := (input.Page - 1) * input.Limit

	var alerts []models.Alert
	if err := q.Order(fmt.Sprintf("%s %s", sort, order)).
		Offset(offset).Limit(input.Limit).
		Find(&alerts).Error; err != nil {
		return nil, huma.Error500InternalServerError("database error")
	}

	out := &ListAlertsOutput{}
	out.Body.Total = total
	out.Body.Page = input.Page
	out.Body.Limit = input.Limit
	out.Body.Alerts = alerts
	return out, nil
}

func getAlert(db *gorm.DB, id string) (*GetAlertOutput, error) {
	var alert models.Alert
	if err := db.Unscoped().First(&alert, "id = ?", id).Error; err != nil {
		return nil, huma.Error404NotFound("alert not found")
	}
	return &GetAlertOutput{Body: alert}, nil
}

func getAlertStats(db *gorm.DB) (*AlertStatsOutput, error) {
	out := &AlertStatsOutput{}

	db.Model(&models.Alert{}).Count(&out.Body.TotalActive) //nolint:errcheck

	type kv struct {
		Key   string
		Count int64
	}

	out.Body.BySeverity = make(map[string]int64)
	var bySev []kv
	db.Model(&models.Alert{}).Select("severity as key, count(*) as count").Group("severity").Scan(&bySev) //nolint:errcheck
	for _, r := range bySev {
		out.Body.BySeverity[r.Key] = r.Count
	}

	out.Body.ByStatus = make(map[string]int64)
	var byStatus []kv
	db.Model(&models.Alert{}).Select("status as key, count(*) as count").Group("status").Scan(&byStatus) //nolint:errcheck
	for _, r := range byStatus {
		out.Body.ByStatus[r.Key] = r.Count
	}

	out.Body.ByFeed = make(map[string]int64)
	var byFeed []kv
	db.Model(&models.Alert{}).Select("feed_source as key, count(*) as count").Group("feed_source").Scan(&byFeed) //nolint:errcheck
	for _, r := range byFeed {
		out.Body.ByFeed[r.Key] = r.Count
	}

	db.Model(&models.Alert{}).Where("forwarded = ?", true).Count(&out.Body.Forwarded) //nolint:errcheck

	var destCount int64
	db.Model(&models.XMPPPeer{}).Where("enabled = ?", true).Count(&destCount) //nolint:errcheck
	out.Body.DestinationsConfigured = destCount > 0
	if out.Body.DestinationsConfigured {
		db.Model(&models.Alert{}).Where("forwarded = ?", false).Count(&out.Body.PendingForward) //nolint:errcheck
	}

	return out, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
