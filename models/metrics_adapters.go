package models

import (
	"time"

	"github.com/kiali/kiali/kubernetes"
)

// GraphQuery indeed
type GraphQuery struct {
	Time              time.Time
	Duration          time.Duration
	Namespace         string
	GraphAdapter      string
	AggregationLevel  string
	IntermediateNodes []string
}

// TitleAndName :)
type TitleAndName struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

// AdaptersInfo holds some info about adapters
type AdaptersInfo struct {
	List  []TitleAndName `json:"list"`
	First *GraphResponse `json:"first"`
}

// GraphResponse :)
type GraphResponse struct {
	Adapter kubernetes.GraphAdapterSpec `json:"adapter"`
	Edges   []Edge                      `json:"edges"`
}

type Edge struct {
	SourceID string      `json:"sourceID"`
	DestID   string      `json:"destID"`
	Labels   []EdgeLabel `json:"labels"`
}

type EdgeLabel struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}
