package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"clortho/internal/models"
)


// ParseExpirationDuration parses a duration string like "3d", "2w", "1mo", "1y"
// and returns the expiration time from now.
func ParseExpirationDuration(d string) (time.Time, error) {
	if len(d) < 2 {
		return time.Time{}, fmt.Errorf("duration too short")
	}

	var unit string
	var valStr string
	if strings.HasSuffix(d, "mo") {
		unit = "mo"
		valStr = d[:len(d)-2]
	} else {
		unit = d[len(d)-1:]
		valStr = d[:len(d)-1]
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number")
	}

	now := time.Now()
	switch unit {
	case "m":
		return now.Add(time.Minute * time.Duration(val)), nil
	case "h":
		return now.Add(time.Hour * time.Duration(val)), nil
	case "d":
		return now.AddDate(0, 0, val), nil
	case "w":
		return now.AddDate(0, 0, val*7), nil
	case "mo":
		return now.AddDate(0, val, 0), nil
	case "y":
		return now.AddDate(val, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unknown unit %q", unit)
	}
}

// ParsePaginationParams extracts page and limit from query parameters
func ParsePaginationParams(c *gin.Context) models.PaginationParams {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	
	// Enforce a sensible max limit to prevent abuse
	if limit > 1000 {
		limit = 1000
	}

	return models.PaginationParams{
		Page:  page,
		Limit: limit,
	}
}
