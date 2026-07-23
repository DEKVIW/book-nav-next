package domain

// CanViewWebsite implements private-link visibility without string-contains bugs.
func CanViewWebsite(site *Website, user *User, viewerIDs []int64) bool {
	if site == nil {
		return false
	}
	if !site.IsPrivate {
		return true
	}
	if user == nil {
		return false
	}
	if user.Role.IsAdmin() {
		return true
	}
	if site.CreatedBy == user.ID {
		return true
	}
	for _, id := range viewerIDs {
		if id == user.ID {
			return true
		}
	}
	return false
}

func CanEditWebsite(site *Website, user *User) bool {
	if user == nil || site == nil {
		return false
	}
	if user.Role.IsAdmin() {
		return true
	}
	return site.CreatedBy == user.ID
}
