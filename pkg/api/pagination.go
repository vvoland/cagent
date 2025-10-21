package api

import (
	"fmt"
	"strconv"

	"github.com/docker/cagent/pkg/session"
)

type PaginationParams struct {
	Limit  int
	Before string
	After  string
}

const DefaultLimit = 50

const MaxLimit = 200

func PaginateMessages(messages []session.Message, params PaginationParams) ([]session.Message, *PaginationMetadata, error) {
	totalCount := len(messages)

	limit := params.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	var beforeIndex, afterIndex int
	var err error

	if params.Before != "" {
		beforeIndex, err = strconv.Atoi(params.Before)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid before cursor: %w", err)
		}
	}

	if params.After != "" {
		afterIndex, err = strconv.Atoi(params.After)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid after cursor: %w", err)
		}
	}

	startIdx := 0
	endIdx := totalCount

	if params.After != "" {
		startIdx = afterIndex + 1
		if startIdx >= totalCount {
			return []session.Message{}, &PaginationMetadata{
				TotalMessages: totalCount,
				Limit:         0,
				HasMore:       false,
			}, nil
		}
	}

	if params.Before != "" {
		endIdx = beforeIndex
		if endIdx <= 0 {
			return []session.Message{}, &PaginationMetadata{
				TotalMessages: totalCount,
				Limit:         0,
				HasMore:       false,
			}, nil
		}
	}

	if params.Before != "" {
		actualStart := max(endIdx-limit, startIdx)
		startIdx = actualStart
	} else {
		actualEnd := min(startIdx+limit, endIdx)
		endIdx = actualEnd
	}

	paginatedMessages := messages[startIdx:endIdx]

	metadata := &PaginationMetadata{
		TotalMessages: totalCount,
		Limit:         len(paginatedMessages),
		HasMore:       false,
	}

	if params.Before != "" {
		metadata.HasMore = startIdx > 0
	} else {
		metadata.HasMore = endIdx < totalCount
	}

	if len(paginatedMessages) > 0 {
		lastIdx := endIdx - 1
		metadata.NextCursor = strconv.Itoa(lastIdx)
		metadata.PrevCursor = strconv.Itoa(startIdx)
	}

	return paginatedMessages, metadata, nil
}
