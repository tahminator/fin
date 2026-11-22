package mcpserver

import (
	"context"
	"errors"
	"math"
	"slices"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
)

type RefreshDataInput struct {
	ConnectionIDs []string `json:"connection_ids,omitempty" jsonschema:"Banks to refresh (IDs from list_connections). Defaults to all of them."`
}

type RefreshResult struct {
	ConnectionID      string  `json:"connection_id"`
	Institution       string  `json:"institution"`
	Result            string  `json:"result" jsonschema:"synced, throttled (refreshed recently; see retry_after_seconds), needs_reauth (the person must reconnect it in the fin app) or failed"`
	RetryAfterSeconds *int    `json:"retry_after_seconds,omitempty"`
	LastSyncedAt      *string `json:"last_synced_at"`
}

type RefreshDataOutput struct {
	Results     []RefreshResult `json:"results"`
	Synced      int             `json:"synced"`
	Throttled   int             `json:"throttled"`
	NeedsReauth int             `json:"needs_reauth"`
	Failed      int             `json:"failed"`
}

func (t *toolset) refreshData(ctx context.Context, input RefreshDataInput) (RefreshDataOutput, error) {
	if !t.grant.Has(domain.ScopeSync) {
		return RefreshDataOutput{}, invalidParams("this app was not allowed to refresh data")
	}

	wanted, err := parseIDs("connection_ids", input.ConnectionIDs)
	if err != nil {
		return RefreshDataOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()

	connections, _, err := t.deps.Connections.ListByUser(ctx, t.userID(), maxListedRecords, 0)
	if err != nil {
		return RefreshDataOutput{}, err
	}

	output := RefreshDataOutput{Results: []RefreshResult{}}
	attempted := map[uuid.UUID]string{}
	retryAfter := map[uuid.UUID]int{}

	for _, connection := range connections {
		if len(wanted) > 0 && !slices.Contains(wanted, connection.ID) {
			continue
		}

		result, wait := t.refreshOne(ctx, connection)
		attempted[connection.ID] = result

		if wait > 0 {
			retryAfter[connection.ID] = wait
		}
	}

	for _, id := range wanted {
		if _, ok := attempted[id]; !ok {
			return RefreshDataOutput{}, invalidParams("connection %s was not found; use list_connections", id)
		}
	}

	refreshed, _, err := t.deps.Connections.ListByUser(context.WithoutCancel(ctx), t.userID(), maxListedRecords, 0)
	if err != nil {
		return RefreshDataOutput{}, err
	}

	for _, connection := range refreshed {
		result, ok := attempted[connection.ID]
		if !ok {
			continue
		}

		entry := RefreshResult{
			ConnectionID: connection.ID.String(),
			Institution:  institutionName(connection.InstitutionName),
			Result:       result,
			LastSyncedAt: formatOptionalTime(connection.LastSyncedAt),
		}

		if wait, ok := retryAfter[connection.ID]; ok {
			entry.RetryAfterSeconds = &wait
		}

		output.Results = append(output.Results, entry)
		output.count(result)
	}

	return output, nil
}

func (t *toolset) refreshOne(ctx context.Context, connection domain.Connection) (string, int) {
	if connection.Status == domain.ConnectionStatusNeedsReauth {
		return "needs_reauth", 0
	}

	if ctx.Err() != nil {
		return "failed", 0
	}

	err := t.deps.Syncer.Sync(ctx, t.userID(), connection.ID, metrics.SyncTriggerMCP)

	var throttled *domain.ThrottledError

	switch {
	case err == nil:
		return "synced", 0
	case errors.As(err, &throttled):
		return "throttled", max(int(math.Ceil(throttled.RetryAfter.Seconds())), 1)
	case errors.Is(err, domain.ErrReauthRequired):
		return "needs_reauth", 0
	}

	return "failed", 0
}

func (o *RefreshDataOutput) count(result string) {
	switch result {
	case "synced":
		o.Synced++
	case "throttled":
		o.Throttled++
	case "needs_reauth":
		o.NeedsReauth++
	default:
		o.Failed++
	}
}
