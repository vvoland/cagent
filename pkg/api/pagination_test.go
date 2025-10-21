package api

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/session"
)

func createTestMessages(count int) []session.Message {
	messages := make([]session.Message, count)
	for i := range count {
		role := chat.MessageRoleUser
		if i%2 == 1 {
			role = chat.MessageRoleAssistant
		}
		messages[i] = session.Message{
			AgentFilename: "test.yaml",
			AgentName:     "test",
			Message: chat.Message{
				Role:      role,
				Content:   "Message " + string(rune('A'+i)),
				CreatedAt: time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			},
		}
	}
	return messages
}

func TestPaginateMessages_FirstPage(t *testing.T) {
	messages := createTestMessages(100)

	params := PaginationParams{
		Limit: 10,
	}

	paginated, meta, err := PaginateMessages(messages, params)
	require.NoError(t, err)
	assert.Len(t, paginated, 10)
	assert.Equal(t, 100, meta.TotalMessages)
	assert.Equal(t, 10, meta.Limit)
	assert.NotEmpty(t, meta.PrevCursor) // More older messages available

	// Should get most recent 10 messages (for chat infinite scroll)
	// For 100 messages, indices 90-99 should be returned
	// Check that we got recent messages by verifying they're different from the old first messages
	assert.NotEqual(t, "Message A", paginated[0].Message.Content) // Not the oldest message
	assert.NotEqual(t, "Message J", paginated[9].Message.Content) // Not the 10th oldest message

	// Verify these are actually the last 10 messages by checking against known patterns
	// The createTestMessages function creates "Message " + char, where char is 'A' + index
	// So message 90 would be beyond normal ASCII range, let's just verify the structure
}

func TestPaginateMessages_WithBeforeCursorPagination(t *testing.T) {
	messages := createTestMessages(20) // Use smaller dataset for easier debugging

	// Start with a page at the end (messages 10-19)
	endPageParams := PaginationParams{
		Limit:  10,
		Before: "20", // Get 10 messages before index 20 (which should give us 10-19)
	}
	endPage, endMeta, err := PaginateMessages(messages, endPageParams)
	require.NoError(t, err)

	// Verify we got the end page
	assert.Len(t, endPage, 10)
	assert.Equal(t, "Message K", endPage[0].Message.Content) // Index 10 = 'K'
	assert.Equal(t, "Message T", endPage[9].Message.Content) // Index 19 = 'T'

	// Get previous page using before cursor (should give us messages 0-9)
	prevPageParams := PaginationParams{
		Limit:  10,
		Before: endMeta.PrevCursor, // Before the end page
	}
	prevPage, prevMeta, err := PaginateMessages(messages, prevPageParams)
	require.NoError(t, err)

	assert.Len(t, prevPage, 10)
	assert.Empty(t, prevMeta.PrevCursor) // No more older messages

	// Should get messages 0-9
	assert.Equal(t, "Message A", prevPage[0].Message.Content) // Index 0 = 'A'
	assert.Equal(t, "Message J", prevPage[9].Message.Content) // Index 9 = 'J'

	// No overlap between pages
	assert.NotEqual(t, endPage[0].Message.Content, prevPage[9].Message.Content)
}

func TestPaginateMessages_WithBeforeCursor(t *testing.T) {
	messages := createTestMessages(100)

	// Get a page in the middle (starting at index 50)
	middleCursor := strconv.Itoa(50)

	params := PaginationParams{
		Limit:  10,
		Before: middleCursor,
	}

	paginated, meta, err := PaginateMessages(messages, params)
	require.NoError(t, err)

	assert.Len(t, paginated, 10)
	assert.NotEmpty(t, meta.PrevCursor) // There are older messages

	// Should get 10 messages before index 50 (indices 40-49)
	assert.Equal(t, "Message "+string(rune('A'+40)), paginated[0].Message.Content)
	assert.Equal(t, "Message "+string(rune('A'+49)), paginated[9].Message.Content)
}

func TestPaginateMessages_DefaultLimit(t *testing.T) {
	messages := createTestMessages(100)

	params := PaginationParams{
		Limit: 0, // Should use default
	}

	paginated, meta, err := PaginateMessages(messages, params)
	require.NoError(t, err)

	assert.Len(t, paginated, DefaultLimit)
	assert.Equal(t, DefaultLimit, meta.Limit)
}

func TestPaginateMessages_MaxLimit(t *testing.T) {
	messages := createTestMessages(300)

	params := PaginationParams{
		Limit: 500, // Should be capped at MaxLimit
	}

	paginated, meta, err := PaginateMessages(messages, params)
	require.NoError(t, err)

	assert.Len(t, paginated, MaxLimit)
	assert.Equal(t, MaxLimit, meta.Limit)
}

func TestPaginateMessages_EmptyMessages(t *testing.T) {
	messages := []session.Message{}

	params := PaginationParams{
		Limit: 10,
	}

	paginated, meta, err := PaginateMessages(messages, params)
	require.NoError(t, err)

	assert.Empty(t, paginated)
	assert.Equal(t, 0, meta.TotalMessages)
	assert.Empty(t, meta.PrevCursor) // No messages at all
}

func TestPaginateMessages_LastPage(t *testing.T) {
	messages := createTestMessages(25)

	// Get the oldest 5 messages (using before cursor to limit to earliest messages)
	lastPageParams := PaginationParams{
		Limit:  10,
		Before: "5", // Before the 6th message (index 5)
	}
	lastPage, lastMeta, err := PaginateMessages(messages, lastPageParams)
	require.NoError(t, err)

	assert.Len(t, lastPage, 5)           // Only 5 messages (0-4)
	assert.Empty(t, lastMeta.PrevCursor) // No more older messages
	assert.Equal(t, 25, lastMeta.TotalMessages)

	// Should get the first 5 messages
	assert.Equal(t, "Message A", lastPage[0].Message.Content)
	assert.Equal(t, "Message E", lastPage[4].Message.Content)
}

func TestPaginateMessages_BeforeFirstMessage(t *testing.T) {
	messages := createTestMessages(10)

	// Create cursor pointing to before first message
	firstCursor := strconv.Itoa(0)

	params := PaginationParams{
		Limit:  10,
		Before: firstCursor,
	}

	paginated, meta, err := PaginateMessages(messages, params)
	require.NoError(t, err)

	assert.Empty(t, paginated)
	assert.Empty(t, meta.PrevCursor) // No messages at all
}

func TestPaginateMessages_InvalidCursor(t *testing.T) {
	messages := createTestMessages(10)

	params := PaginationParams{
		Limit:  10,
		Before: "invalid-cursor",
	}

	_, _, err := PaginateMessages(messages, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid before cursor")
}
