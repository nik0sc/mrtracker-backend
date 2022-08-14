package smrt

import (
	"strconv"

	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/model"
)

const (
	fakeUA = "SMRT Connect/3.3.3 Android/9.0"
)

type NextTrains struct {
	// Common data

	// Station code (line-digit), for interchanges comma separated
	Code string `json:"code,omitempty"`
	// Friendly name of station
	Mrt string `json:"mrt,omitempty"`

	// Platform-specific

	// Next train time: either a numeric string, "Arr", or "N/A"
	NextTrainArr string `json:"next_train_arr,omitempty"`
	// Train's destination (always present even if time is "N/A"), or "Do not board"
	NextTrainDestination string `json:"next_train_destination,omitempty"`
	// Platform ID: three-letter code followed by platform letter eg "CTH_A"
	PlatformID string `json:"platform_ID,omitempty"`
	// Always 1, except when it isn't
	Status int `json:"status,omitempty"`
	// Train following the next train
	SubseqTrainArr string `json:"subseq_train_arr,omitempty"`
	// May not always be present
	SubseqTrainDestination string `json:"subseq_train_destination,omitempty"`
}

func arrToInt(a string) int {
	if a == "Arr" {
		return 0
	} else if i, err := strconv.Atoi(a); err != nil {
		return i
	} else {
		return -1
	}
}

func (p *NextTrains) Valid() bool {
	// If station was not found the platform id is empty
	return p != nil && p.PlatformID != ""
}

func (p *NextTrains) ToModelPlatform() model.Platform {
	// model package should not have knowledge of specific data source implementation
	return model.Platform{
		Next:  arrToInt(p.NextTrainArr),
		Dest:  p.NextTrainDestination,
		Next2: arrToInt(p.SubseqTrainArr),
		Dest2: p.SubseqTrainDestination,
	}
}

type Result []NextTrains

func StationResultToModel(r map[string]Result, src data.Line) model.Line {
	out := make(model.Line, len(src))

	for i := 0; i < len(src); i++ {
		results, ok := r[src[i].Name]
		if !ok {
			continue
		}

		platformID := src[i].PlatformID()
		for _, r := range results {
			if r.PlatformID == platformID {
				out[i] = r.ToModelPlatform()
				break
			}
		}
	}

	return out
}

func PlatformResultToModel(r map[string]*NextTrains, src data.Line) model.Line {
	out := make(model.Line, len(src))

	for i := 0; i < len(src); i++ {
		r, ok := r[src[i].Platform]
		if !ok {
			continue
		}
		out[i] = r.ToModelPlatform()
	}

	return out
}
