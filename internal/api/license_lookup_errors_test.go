package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
	"clortho/internal/store"
)

func TestCheckLicenseLookupErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"missing", store.ErrNotFound, http.StatusNotFound, `{"error":"License not found"}`},
		{"wrapped missing", fmt.Errorf("lookup: %w", store.ErrNotFound), http.StatusNotFound, `{"error":"License not found"}`},
		{"database unavailable", errors.New("connection refused at internal-db:5432"), http.StatusServiceUnavailable, `{"error":"License validation service unavailable"}`},
		{"deadline exceeded", context.DeadlineExceeded, http.StatusServiceUnavailable, `{"error":"License validation service unavailable"}`},
		{"cancelled", context.Canceled, http.StatusServiceUnavailable, `{"error":"License validation service unavailable"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			licenses := new(MockLicenseStore)
			licenses.On("GetLicenseByKey", mock.Anything, "TEST-lookup").Return(nil, tc.err).Once()
			logs := new(MockLogStore)
			recorded := make(chan *models.LicenseCheckLog, 1)
			logs.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) { recorded <- args.Get(1).(*models.LicenseCheckLog) }).Return(nil).Once()
			router := gin.New()
			router.GET("/check", handlers.CheckLicenseHandler(licenses, new(MockProductStore), "", logs))
			req := httptest.NewRequest(http.MethodGet, "/check", nil)
			req.Header.Set("X-License-Key", "TEST-lookup")
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			assert.Equal(t, tc.status, res.Code)
			assert.JSONEq(t, tc.body, res.Body.String())
			if tc.status == http.StatusServiceUnavailable {
				assert.Equal(t, "30", res.Header().Get("Retry-After"))
			}
			select {
			case entry := <-recorded:
				assert.Equal(t, tc.status, entry.StatusCode)
				payload, err := json.Marshal(entry.ResponsePayload)
				require.NoError(t, err)
				assert.JSONEq(t, tc.body, string(payload))
			case <-time.After(time.Second):
				t.Fatal("license check was not logged")
			}
			licenses.AssertExpectations(t)
		})
	}
}
