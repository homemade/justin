package justin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/homemade/justin/api"
)

func paginatedFundraisingPagesForEvent(svc *Service, eventID uint, pagination uint) (results []*FundraisingPageRef, totalPagination uint, totalFundraisingPages uint, err error) {

	method := "GET"
	path := bytes.NewBuffer([]byte(svc.BasePath))
	path.WriteString("/")
	path.WriteString(svc.APIKey)
	path.WriteString("/v1/event/")
	path.WriteString(strconv.FormatUint(uint64(eventID), 10))
	path.WriteString("/pages/?pageSize=100")

	// set pagination
	pg := "1"
	if pagination > 0 {
		pg = strconv.FormatUint(uint64(pagination), 10)
	}

	req, err := api.BuildRequest(UserAgent, ContentType, method, path.String()+"&page="+pg, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	res, resBody, err := api.Do(svc.client, svc.origin, "FundraisingPagesForEvent", req, "", svc.HTTPLogger)
	if err != nil {
		return nil, 0, 0, err
	}

	if res.StatusCode != 200 {
		return nil, 0, 0, fmt.Errorf("invalid response %s", res.Status)
	}
	type page struct {
		CharityID     uint   `json:"charityId"`
		PageID        uint   `json:"pageId"`
		PageShortName string `json:"pageShortName"`
	}
	var result = struct {
		TotalPagination       uint   `json:"totalPages"`
		TotalFundraisingPages uint   `json:"totalFundraisingPages"`
		FundraisingPages      []page `json:"fundraisingPages"`
	}{}

	if err := json.Unmarshal([]byte(resBody), &result); err != nil {
		return nil, 0, 0, fmt.Errorf("invalid response %v", err)
	}

	for _, p := range result.FundraisingPages {
		if p.PageID > 0 {
			results = append(results, &FundraisingPageRef{
				charityID: p.CharityID,
				eventID:   eventID,
				id:        p.PageID,
				shortName: p.PageShortName,
			})

		}
	}

	return results, result.TotalPagination, result.TotalFundraisingPages, nil
}
