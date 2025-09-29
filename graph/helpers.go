package graph

// Legacy helper functions that are still needed for backward compatibility
// These functions have been refactored and moved to appropriate service classes
// but are kept here for any remaining direct usage

// getUserOrDefault returns the user string or "system" if nil
// This is a simple utility function used across resolvers
func getUserOrDefault(user *string) string {
	if user != nil {
		return *user
	}
	return "system"
}