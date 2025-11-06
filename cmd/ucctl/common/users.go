package common

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"userclouds.com/idp"
)

// GetUserIDsByEmail looks up all users with the specified email address and returns their IDs.
// Returns an error if no users are found with the email.
func GetUserIDsByEmail(ctx context.Context, mgmtClient *idp.ManagementClient, email string) ([]uuid.UUID, error) {
	// List all users and filter by email
	allUsers, err := mgmtClient.ListUserBaseProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Filter by email manually
	var userIDs []uuid.UUID
	for _, user := range allUsers.Data {
		if user.Email == email {
			userID, err := uuid.FromString(user.ID)
			if err != nil {
				return nil, fmt.Errorf("invalid user ID: %w", err)
			}
			userIDs = append(userIDs, userID)
		}
	}

	if len(userIDs) == 0 {
		return nil, fmt.Errorf("no user found with email %s", email)
	}

	return userIDs, nil
}

// GetSingleUserIDByEmail looks up a user by email and returns their ID.
// Returns an error if no users are found or if multiple users have the same email.
func GetSingleUserIDByEmail(ctx context.Context, mgmtClient *idp.ManagementClient, email string) (uuid.UUID, error) {
	userIDs, err := GetUserIDsByEmail(ctx, mgmtClient, email)
	if err != nil {
		return uuid.Nil, err
	}

	if len(userIDs) > 1 {
		return uuid.Nil, fmt.Errorf("multiple users found with email: %s, please use --user-id instead", email)
	}

	return userIDs[0], nil
}

// GetUsersByEmail looks up all users with the specified email address and returns their profiles.
// Returns an error if no users are found with the email.
func GetUsersByEmail(ctx context.Context, mgmtClient *idp.ManagementClient, email string) ([]idp.UserBaseProfileResponse, error) {
	// List all users and filter by email
	allUsers, err := mgmtClient.ListUserBaseProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Filter by email manually
	var matchingUsers []idp.UserBaseProfileResponse
	for _, user := range allUsers.Data {
		if user.Email == email {
			matchingUsers = append(matchingUsers, user)
		}
	}

	if len(matchingUsers) == 0 {
		return nil, fmt.Errorf("no user found with email %s", email)
	}

	return matchingUsers, nil
}
