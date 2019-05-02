package github

import (
	"encoding/json"
	"strconv"
	"time"
)

type EntityID string

func (id *EntityID) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}

	var sid string
	if err := json.Unmarshal(b, &sid); err != nil {
		var nid int
		if err := json.Unmarshal(b, &nid); err != nil {
			return err
		}
		sid = strconv.Itoa(nid)
	}

	*id = EntityID(sid)

	return nil
}

type BaseEntity struct {
	ID     EntityID `json:"id"`
	NodeID string   `json:"node_id,omitempty"`
	URL    string   `json:"url,omitempty"`

	// ···
}

type Repository struct {
	BaseEntity

	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Owner struct {
	BaseEntity

	Login string `json:"login"`
}

type Issue struct {
	BaseEntity

	Number    int       `json:"number"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Label struct {
	BaseEntity

	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

type Project struct {
	BaseEntity

	Name string `json:"name"`
}

type Column struct {
	BaseEntity

	Name string `json:"name"`
}
