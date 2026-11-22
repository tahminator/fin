package handler

import (
	"uuid"

	"github.com/danielgtaylor/huma/v2"
)

type AccountFilterParams struct {
	AccountID        string   `query:"accountId" format:"uuid" doc:"Only data of this account"`
	ExcludeAccountID []string `query:"excludeAccountId,explode" format:"uuid" doc:"Leave out data of these accounts"`
}

type accountFilter struct {
	accountID        *uuid.UUID
	excludeAccountID []uuid.UUID
}

func (p AccountFilterParams) parse() (accountFilter, error) {
	var filter accountFilter

	if p.AccountID != "" {
		accountID, err := uuid.Parse(p.AccountID)
		if err != nil {
			return accountFilter{}, huma.Error422UnprocessableEntity("accountId must be a UUID")
		}

		filter.accountID = &accountID
	}

	excludeAccountIDs, err := parseUUIDs(p.ExcludeAccountID)
	if err != nil {
		return accountFilter{}, huma.Error422UnprocessableEntity("excludeAccountId must be a list of UUIDs")
	}

	filter.excludeAccountID = excludeAccountIDs

	return filter, nil
}

func parseUUIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	parsed := make([]uuid.UUID, len(raw))

	for i, value := range raw {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}

		parsed[i] = id
	}

	return parsed, nil
}
