package api

import (
	"fmt"
	"strconv"

	"github.com/docker/cagent/pkg/session"
)

type PaginationParams struct {
	Limit  int
	Before string
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

	var beforeIndex int
	var err error

	if params.Before != "" {
		beforeIndex, err = strconv.Atoi(params.Before)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid before cursor: %w", err)
		}
	}

	startIdx := 0
	endIdx := totalCount

	if params.Before != "" {
		endIdx = beforeIndex
		if endIdx <= 0 {
			return []session.Message{}, &PaginationMetadata{
				TotalMessages: totalCount,
				Limit:         0,
			}, nil
		}
		actualStart := max(endIdx-limit, startIdx)
		startIdx = actualStart
	} else {
		actualStart := max(totalCount-limit, 0)
		startIdx = actualStart
		endIdx = totalCount
	}

	paginatedMessages := messages[startIdx:endIdx]

	metadata := &PaginationMetadata{
		TotalMessages: totalCount,
		Limit:         len(paginatedMessages),
	}

	// Only set cursor if there are more (older) messages available
	if len(paginatedMessages) > 0 && startIdx > 0 {
		metadata.PrevCursor = strconv.Itoa(startIdx)
	}

	return paginatedMessages, metadata, nil
}
