package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kiali/kiali/models"
)

func GenericGraphAdapters(w http.ResponseWriter, r *http.Request) {
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	params := r.URL.Query()
	q := models.GraphQuery{Namespace: params.Get("namespace")}
	if dur := params.Get("duration"); dur != "" {
		if num, err := strconv.ParseInt(dur, 10, 64); err == nil {
			q.Duration = time.Duration(num) * time.Second
		} else {
			RespondWithError(w, http.StatusBadRequest, "Bad request, cannot parse query parameter 'duration'")
			return
		}
	} else {
		q.Duration = 60 * time.Second
	}
	if t := params.Get("time"); t != "" {
		if num, err := strconv.ParseInt(t, 10, 64); err == nil {
			q.Time = time.Unix(num, 0)
		} else {
			RespondWithError(w, http.StatusBadRequest, "Bad request, cannot parse query parameter 'time'")
			return
		}
	} else {
		q.Time = time.Now()
	}
	adapters, err := business.GenericGraph.GetGraphAdapters(q)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, adapters)
}

func GenericGraph(w http.ResponseWriter, r *http.Request) {
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	params := r.URL.Query()
	q := models.GraphQuery{
		Namespace:        params.Get("namespace"),
		AggregationLevel: params.Get("aggregation"),
		GraphAdapter:     params.Get("adapter"),
	}
	if dur := params.Get("duration"); dur != "" {
		if num, err := strconv.ParseInt(dur, 10, 64); err == nil {
			q.Duration = time.Duration(num) * time.Second
		} else {
			RespondWithError(w, http.StatusBadRequest, "Bad request, cannot parse query parameter 'duration'")
			return
		}
	} else {
		q.Duration = 60 * time.Second
	}
	if t := params.Get("time"); t != "" {
		if num, err := strconv.ParseInt(t, 10, 64); err == nil {
			q.Time = time.Unix(num, 0)
		} else {
			RespondWithError(w, http.StatusBadRequest, "Bad request, cannot parse query parameter 'time'")
			return
		}
	} else {
		q.Time = time.Now()
	}
	graph, err := business.GenericGraph.GetGraph(q)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, graph)
}
