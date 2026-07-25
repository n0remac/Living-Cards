package cardassets

import (
	"regexp"
	"strings"
)

const BackgroundURLPrefix = "/assets/card-backgrounds/"

var backgroundIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

func ValidBackgroundID(assetID string) bool {
	return backgroundIDPattern.MatchString(strings.TrimSpace(assetID))
}

func BackgroundURL(assetID string) string {
	assetID = strings.TrimSpace(assetID)
	if !ValidBackgroundID(assetID) {
		return ""
	}
	return BackgroundURLPrefix + assetID + ".webp"
}
