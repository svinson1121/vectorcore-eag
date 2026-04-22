package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/vectorcore/eag/internal/models"
	"github.com/vectorcore/eag/internal/xmpp"
)

func registerXMPPHandlers(api huma.API, db *gorm.DB, srv *xmpp.Server) {
	huma.Register(api, huma.Operation{
		OperationID: "list-xmpp-peers",
		Method:      http.MethodGet,
		Path:        "/api/v1/xmpp",
		Summary:     "List XMPP peers",
		Tags:        []string{"XMPP"},
	}, func(ctx context.Context, input *struct{}) (*PeerListOutput, error) {
		var peers []models.XMPPPeer
		db.Find(&peers)
		out := &PeerListOutput{}
		out.Body = peersToAPI(peers)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-xmpp-peer",
		Method:        http.MethodPost,
		Path:          "/api/v1/xmpp",
		Summary:       "Create XMPP peer",
		Tags:          []string{"XMPP"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *PeerCreateInput) (*PeerOutput, error) {
		peer, err := peerBodyToModel(input.Body)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		if err := db.Create(&peer).Error; err != nil {
			return nil, huma.Error422UnprocessableEntity("could not create: " + err.Error())
		}
		if peer.Enabled {
			srv.SweepUnforwarded(db)
		}
		out := &PeerOutput{}
		out.Body = peerToAPI(peer)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-xmpp-peer",
		Method:      http.MethodPut,
		Path:        "/api/v1/xmpp/{id}",
		Summary:     "Update XMPP peer",
		Tags:        []string{"XMPP"},
	}, func(ctx context.Context, input *PeerUpdateInput) (*PeerOutput, error) {
		var existing models.XMPPPeer
		if err := db.First(&existing, input.ID).Error; err != nil {
			return nil, huma.Error404NotFound("xmpp peer not found")
		}
		updated, err := peerUpdateBodyToModel(input.Body, existing)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		updated.ID = existing.ID
		updated.CreatedAt = existing.CreatedAt
		if err := db.Save(&updated).Error; err != nil {
			return nil, huma.Error422UnprocessableEntity("could not update: " + err.Error())
		}
		if updated.Enabled {
			srv.SweepUnforwarded(db)
		}
		out := &PeerOutput{}
		out.Body = peerToAPI(updated)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-xmpp-peer",
		Method:        http.MethodDelete,
		Path:          "/api/v1/xmpp/{id}",
		Summary:       "Delete XMPP peer",
		Tags:          []string{"XMPP"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *PeerIDInput) (*struct{}, error) {
		if err := db.Delete(&models.XMPPPeer{}, input.ID).Error; err != nil {
			return nil, huma.Error404NotFound("xmpp peer not found")
		}
		return nil, nil
	})
}

// --- Types ---

type PeerAPIModel struct {
	ID             uint     `json:"id"`
	Name           string   `json:"name"`
	Username       string   `json:"username"`
	Enabled        bool     `json:"enabled"`
	FilterSeverity []string `json:"filter_severity"`
	FilterEvent    []string `json:"filter_event"`
	FilterArea     []string `json:"filter_area"`
	FilterStatus   []string `json:"filter_status"`
}

type PeerBody struct {
	Name           string   `json:"name"`
	Username       string   `json:"username"`
	Password       string   `json:"password"`
	Enabled        bool     `json:"enabled"`
	FilterSeverity []string `json:"filter_severity"`
	FilterEvent    []string `json:"filter_event"`
	FilterArea     []string `json:"filter_area"`
	FilterStatus   []string `json:"filter_status"`
}

type PeerUpdateBody struct {
	Name           string   `json:"name"`
	Username       string   `json:"username"`
	Password       *string  `json:"password,omitempty"`
	Enabled        bool     `json:"enabled"`
	FilterSeverity []string `json:"filter_severity"`
	FilterEvent    []string `json:"filter_event"`
	FilterArea     []string `json:"filter_area"`
	FilterStatus   []string `json:"filter_status"`
}

type PeerListOutput struct {
	Body []PeerAPIModel
}

type PeerOutput struct {
	Body PeerAPIModel
}

type PeerIDInput struct {
	ID uint `path:"id"`
}

type PeerCreateInput struct {
	Body PeerBody
}

type PeerUpdateInput struct {
	ID   uint `path:"id"`
	Body PeerUpdateBody
}

func peerBodyToModel(b PeerBody) (models.XMPPPeer, error) {
	marshal := func(v []string) string {
		if len(v) == 0 {
			return "[]"
		}
		out, _ := json.Marshal(v)
		return string(out)
	}
	return models.XMPPPeer{
		Name:           b.Name,
		Username:       b.Username,
		Password:       b.Password,
		Enabled:        b.Enabled,
		FilterSeverity: marshal(b.FilterSeverity),
		FilterEvent:    marshal(b.FilterEvent),
		FilterArea:     marshal(b.FilterArea),
		FilterStatus:   marshal(b.FilterStatus),
	}, nil
}

func peerUpdateBodyToModel(b PeerUpdateBody, existing models.XMPPPeer) (models.XMPPPeer, error) {
	peer, err := peerBodyToModel(PeerBody{
		Name:           b.Name,
		Username:       b.Username,
		Enabled:        b.Enabled,
		FilterSeverity: b.FilterSeverity,
		FilterEvent:    b.FilterEvent,
		FilterArea:     b.FilterArea,
		FilterStatus:   b.FilterStatus,
	})
	if err != nil {
		return models.XMPPPeer{}, err
	}
	peer.Password = existing.Password
	if b.Password != nil && *b.Password != "" {
		peer.Password = *b.Password
	}
	return peer, nil
}

func peerToAPI(p models.XMPPPeer) PeerAPIModel {
	unmarshal := func(s string) []string {
		var v []string
		json.Unmarshal([]byte(s), &v) //nolint:errcheck
		if v == nil {
			return []string{}
		}
		return v
	}
	return PeerAPIModel{
		ID:             p.ID,
		Name:           p.Name,
		Username:       p.Username,
		Enabled:        p.Enabled,
		FilterSeverity: unmarshal(p.FilterSeverity),
		FilterEvent:    unmarshal(p.FilterEvent),
		FilterArea:     unmarshal(p.FilterArea),
		FilterStatus:   unmarshal(p.FilterStatus),
	}
}

func peersToAPI(peers []models.XMPPPeer) []PeerAPIModel {
	out := make([]PeerAPIModel, len(peers))
	for i, p := range peers {
		out[i] = peerToAPI(p)
	}
	return out
}
