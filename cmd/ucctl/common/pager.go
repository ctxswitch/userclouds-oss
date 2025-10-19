package common

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
)

// PageableResult represents a result that can be paginated
type PageableResult[T any] struct {
	Data       []T
	NextCursor string
	HasMore    bool
}

// FetchFunc is a function that fetches a page of results
type FetchFunc[T any] func(ctx context.Context, cursor string, limit int) ([]T, string, error)

// DisplayFunc is a function that displays a page of results with header
type DisplayFunc[T any] func(items []T)

// DisplayWithoutHeaderFunc is a function that displays a page of results without header
type DisplayWithoutHeaderFunc[T any] func(items []T)

// PagerConfig configures the interactive pager
type PagerConfig[T any] struct {
	Ctx                  context.Context
	FetchFunc            FetchFunc[T]
	DisplayFunc          DisplayFunc[T]
	DisplayWithoutHeader DisplayWithoutHeaderFunc[T]
	InitialCursor        string
	NoItemsMessage       string
	ItemName             string // e.g., "object types", "edges"
}

// clearLine clears the current line
func clearLine() {
	fmt.Print("\033[2K\r")
}

// RunInteractivePager runs an interactive paging session
func RunInteractivePager[T any](config PagerConfig[T]) error {
	// Get terminal size
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to a reasonable default
		height = 24
	}

	// Reserve lines for header, footer, and prompts
	pageSize := height - 4

	cursor := config.InitialCursor
	reader := bufio.NewReader(os.Stdin)
	isFirstPage := true

	for {
		// Fetch a page of results
		items, nextCursor, err := config.FetchFunc(config.Ctx, cursor, pageSize)
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", config.ItemName, err)
		}

		if len(items) == 0 {
			if isFirstPage {
				fmt.Println(config.NoItemsMessage)
			} else {
				fmt.Println("\nEnd of results.")
			}
			return nil
		}

		// Display the current page (with header on first page only)
		if isFirstPage {
			config.DisplayFunc(items)
			isFirstPage = false
		} else {
			config.DisplayWithoutHeader(items)
		}

		// Show navigation prompt (use inverse colors for visibility like less)
		hasMore := nextCursor != ""
		if hasMore {
			fmt.Print("\033[7m") // Inverse video
			fmt.Printf(":")
			fmt.Print("\033[0m") // Reset
		} else {
			fmt.Print("\033[7m") // Inverse video
			fmt.Printf("(END)")
			fmt.Print("\033[0m") // Reset
		}

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		// Move cursor up one line and clear it (to remove prompt and the newline from ENTER)
		fmt.Print("\033[1A") // Move up
		clearLine()          // Clear the line

		input = strings.TrimSpace(strings.ToLower(input))

		// Handle user input
		if input == "q" || input == "quit" {
			return nil
		}

		if !hasMore {
			return nil
		}

		// Move to next page
		cursor = nextCursor
	}
}

// RunNonInteractivePager displays all results without pagination
func RunNonInteractivePager[T any](config PagerConfig[T]) error {
	cursor := config.InitialCursor
	isFirstPage := true
	limit := 100

	for {
		items, nextCursor, err := config.FetchFunc(config.Ctx, cursor, limit)
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", config.ItemName, err)
		}

		if len(items) == 0 {
			if isFirstPage {
				fmt.Println(config.NoItemsMessage)
			}
			break
		}

		// Display the results (with header on first page only)
		if isFirstPage {
			config.DisplayFunc(items)
			isFirstPage = false
		} else {
			config.DisplayWithoutHeader(items)
		}

		// If there's no next cursor, we've fetched everything
		if nextCursor == "" {
			break
		}

		cursor = nextCursor
	}

	return nil
}

// TabularDisplay helps display tabular data consistently
type TabularDisplay struct {
	writer *tabwriter.Writer
}

// NewTabularDisplay creates a new tabular display helper
func NewTabularDisplay() *TabularDisplay {
	return &TabularDisplay{
		writer: tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0),
	}
}

// WriteHeader writes a header line
func (t *TabularDisplay) WriteHeader(columns ...string) {
	fmt.Fprintln(t.writer, strings.Join(columns, "\t"))
}

// WriteRow writes a data row
func (t *TabularDisplay) WriteRow(values ...string) {
	fmt.Fprintln(t.writer, strings.Join(values, "\t"))
}

// Flush flushes the writer
func (t *TabularDisplay) Flush() {
	t.writer.Flush()
}

// Filter key constants for different resource types
var (
	EdgeFilterKeys       = []string{"id", "source_object_id", "target_object_id", "created", "updated"}
	EdgeTypeFilterKeys   = []string{"id", "type_name", "organization_id", "source_object_type_id", "target_object_type_id", "created", "updated"}
	ObjectFilterKeys     = []string{"id", "alias", "organization_id", "type_id", "created", "updated"}
	ObjectTypeFilterKeys = []string{"id", "type_name", "created", "updated"}
)

// filterToken represents a token in the filter expression
type filterToken struct {
	typ   string // "field", "and", "or", "lparen", "rparen"
	value string
}

// tokenizeFilter breaks the filter string into tokens
func tokenizeFilter(input string) ([]filterToken, error) {
	var tokens []filterToken
	var current strings.Builder
	inField := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '&':
			if current.Len() > 0 {
				tokens = append(tokens, filterToken{typ: "field", value: strings.TrimSpace(current.String())})
				current.Reset()
				inField = false
			}
			tokens = append(tokens, filterToken{typ: "and", value: "&"})

		case '|':
			if current.Len() > 0 {
				tokens = append(tokens, filterToken{typ: "field", value: strings.TrimSpace(current.String())})
				current.Reset()
				inField = false
			}
			tokens = append(tokens, filterToken{typ: "or", value: "|"})

		case '(':
			if current.Len() > 0 {
				return nil, fmt.Errorf("unexpected characters before '('")
			}
			tokens = append(tokens, filterToken{typ: "lparen", value: "("})

		case ')':
			if current.Len() > 0 {
				tokens = append(tokens, filterToken{typ: "field", value: strings.TrimSpace(current.String())})
				current.Reset()
				inField = false
			}
			tokens = append(tokens, filterToken{typ: "rparen", value: ")"})

		case ' ', '\t':
			// Skip whitespace unless we're inside a field value
			if inField {
				current.WriteByte(ch)
			}

		case '=':
			inField = true
			current.WriteByte(ch)

		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, filterToken{typ: "field", value: strings.TrimSpace(current.String())})
	}

	return tokens, nil
}

// parseFilterExpression recursively parses filter tokens into a filter string
func parseFilterExpression(tokens []filterToken, validKeys []string) (string, int, error) {
	if len(tokens) == 0 {
		return "", 0, fmt.Errorf("unexpected end of expression")
	}

	// Parse primary expression (field or grouped expression)
	var left string
	var consumed int

	if tokens[0].typ == "lparen" {
		// Parse grouped expression
		result, n, err := parseFilterExpression(tokens[1:], validKeys)
		if err != nil {
			return "", 0, err
		}
		consumed = n + 1

		// Expect closing paren
		if consumed >= len(tokens) || tokens[consumed].typ != "rparen" {
			return "", 0, fmt.Errorf("missing closing parenthesis")
		}
		consumed++
		left = result

	} else if tokens[0].typ == "field" {
		// Parse field=value
		parts := strings.SplitN(tokens[0].value, "=", 2)
		if len(parts) != 2 {
			return "", 0, fmt.Errorf("invalid filter format '%s': expected key=value", tokens[0].value)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" || value == "" {
			return "", 0, fmt.Errorf("invalid filter format '%s': key and value cannot be empty", tokens[0].value)
		}

		// Validate the key is a supported field
		valid := false
		for _, validKey := range validKeys {
			if key == validKey {
				valid = true
				break
			}
		}
		if !valid {
			return "", 0, fmt.Errorf("unsupported filter field '%s': must be one of %v", key, validKeys)
		}

		left = fmt.Sprintf("('%s',LK,'%s')", key, value)
		consumed = 1

	} else {
		return "", 0, fmt.Errorf("unexpected token: %s", tokens[0].value)
	}

	// Check for binary operator (& or |)
	if consumed < len(tokens) {
		if tokens[consumed].typ == "and" {
			consumed++
			right, n, err := parseFilterExpression(tokens[consumed:], validKeys)
			if err != nil {
				return "", 0, err
			}
			consumed += n
			return fmt.Sprintf("(%s,AND,%s)", left, right), consumed, nil

		} else if tokens[consumed].typ == "or" {
			consumed++
			right, n, err := parseFilterExpression(tokens[consumed:], validKeys)
			if err != nil {
				return "", 0, err
			}
			consumed += n
			return fmt.Sprintf("(%s,OR,%s)", left, right), consumed, nil

		} else if tokens[consumed].typ == "rparen" {
			// Return without consuming the closing paren (let parent handle it)
			return left, consumed, nil
		}
	}

	return left, consumed, nil
}

// FormatFilterString constructs a filter string from a filter expression
// Supports:
//   - key=value for field comparisons
//   - & for AND operations
//   - | for OR operations
//   - () for grouping
// Examples:
//   - "type_name=user"
//   - "type_name=user&id=123"
//   - "type_name=user|type_name=admin"
//   - "(type_name=user|type_name=admin)&organization_id=456"
func FormatFilterString(filterInput string, validKeys []string) (string, error) {
	if filterInput == "" {
		return "", nil
	}

	// Tokenize the input
	tokens, err := tokenizeFilter(filterInput)
	if err != nil {
		return "", err
	}

	if len(tokens) == 0 {
		return "", nil
	}

	// Parse the expression
	result, consumed, err := parseFilterExpression(tokens, validKeys)
	if err != nil {
		return "", err
	}

	// Ensure all tokens were consumed
	if consumed != len(tokens) {
		return "", fmt.Errorf("unexpected tokens after expression")
	}

	return result, nil
}
