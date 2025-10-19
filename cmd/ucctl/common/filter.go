package common

import (
	"fmt"
	"strings"
)

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
