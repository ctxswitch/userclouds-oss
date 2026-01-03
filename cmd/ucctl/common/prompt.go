package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmDeletion prompts the user to confirm a deletion operation
// Returns true if the user confirms (types 'y' or 'yes'), false otherwise
// If autoApprove is true, it returns true immediately without prompting
func ConfirmDeletion(resourceType, resourceID string, autoApprove bool) bool {
	if autoApprove {
		return true
	}

	fmt.Printf("Are you sure you want to delete %s %s? [y/N]: ", resourceType, resourceID)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
