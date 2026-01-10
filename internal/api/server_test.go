package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
)

// MockProductStore is a mock implementation of store.ProductStore
type MockProductStore struct {
	mock.Mock
}

func (m *MockProductStore) ListProducts(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Product, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.Product), args.Int(1), args.Error(2)
}

func (m *MockProductStore) CreateProduct(ctx context.Context, product *models.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductStore) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductStore) UpdateProduct(ctx context.Context, product *models.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductStore) DeleteProduct(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockLicenseStore is a mock implementation of store.LicenseStore
type MockLicenseStore struct {
	mock.Mock
}

func (m *MockLicenseStore) GetLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.License), args.Error(1)
}
func (m *MockLicenseStore) GetLicense(ctx context.Context, id string) (*models.License, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.License), args.Error(1)
}
func (m *MockLicenseStore) CreateLicense(ctx context.Context, license *models.License) error {
	args := m.Called(ctx, license)
	return args.Error(0)
}
func (m *MockLicenseStore) DeleteLicense(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}
func (m *MockLicenseStore) ListLicenses(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.License, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.License), args.Int(1), args.Error(2)
}
func (m *MockLicenseStore) UpdateLicense(ctx context.Context, license *models.License) error {
	args := m.Called(ctx, license)
	return args.Error(0)
}

// MockReleaseStore is a mock implementation of store.ReleaseStore
type MockReleaseStore struct {
	mock.Mock
}

func (m *MockReleaseStore) ListReleasesByProduct(ctx context.Context, productID string, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	args := m.Called(ctx, productID, ownerID, pagination)
	return args.Get(0).([]models.Release), args.Int(1), args.Error(2)
}
func (m *MockReleaseStore) CreateRelease(ctx context.Context, release *models.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}
func (m *MockReleaseStore) UpdateRelease(ctx context.Context, release *models.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}
func (m *MockReleaseStore) DeleteRelease(ctx context.Context, releaseID string) error {
	args := m.Called(ctx, releaseID)
	return args.Error(0)
}

func (m *MockReleaseStore) GetRelease(ctx context.Context, releaseID string) (*models.Release, error) {
	args := m.Called(ctx, releaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Release), args.Error(1)
}

func (m *MockReleaseStore) ListReleasesByProductGroup(ctx context.Context, productGroupID string, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	args := m.Called(ctx, productGroupID, ownerID, pagination)
	return args.Get(0).([]models.Release), args.Int(1), args.Error(2)
}

func (m *MockReleaseStore) ListGlobalReleases(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.Release), args.Int(1), args.Error(2)
}

func (m *MockReleaseStore) ListAllReleases(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.Release), args.Int(1), args.Error(2)
}

type MockFeatureStore struct {
	mock.Mock
}

func (m *MockFeatureStore) ListFeaturesByProduct(ctx context.Context, productID string, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	args := m.Called(ctx, productID, ownerID, pagination)
	return args.Get(0).([]models.Feature), args.Int(1), args.Error(2)
}
func (m *MockFeatureStore) CreateFeature(ctx context.Context, feature *models.Feature) error {
	args := m.Called(ctx, feature)
	return args.Error(0)
}
func (m *MockFeatureStore) UpdateFeature(ctx context.Context, feature *models.Feature) error {
	args := m.Called(ctx, feature)
	return args.Error(0)
}
func (m *MockFeatureStore) DeleteFeature(ctx context.Context, featureID string) error {
	args := m.Called(ctx, featureID)
	return args.Error(0)
}

func (m *MockFeatureStore) GetFeature(ctx context.Context, featureID string) (*models.Feature, error) {
	args := m.Called(ctx, featureID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Feature), args.Error(1)
}

func (m *MockFeatureStore) ListFeaturesByProductGroup(ctx context.Context, productGroupID string, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	args := m.Called(ctx, productGroupID, ownerID, pagination)
	return args.Get(0).([]models.Feature), args.Int(1), args.Error(2)
}

func (m *MockFeatureStore) ListGlobalFeatures(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.Feature), args.Int(1), args.Error(2)
}

func (m *MockFeatureStore) ListAllFeatures(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.Feature), args.Int(1), args.Error(2)
}

// MockProductGroupStore is a mock implementation of store.ProductGroupStore
type MockProductGroupStore struct {
	mock.Mock
}

func (m *MockProductGroupStore) ListProductGroups(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.ProductGroup, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.ProductGroup), args.Int(1), args.Error(2)
}

func (m *MockProductGroupStore) CreateProductGroup(ctx context.Context, group *models.ProductGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockProductGroupStore) GetProductGroup(ctx context.Context, id string) (*models.ProductGroup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductGroup), args.Error(1)
}

func (m *MockProductGroupStore) UpdateProductGroup(ctx context.Context, group *models.ProductGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockProductGroupStore) DeleteProductGroup(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockLogStore is a mock implementation of store.LogStore
type MockLogStore struct {
	mock.Mock
}

func (m *MockLogStore) CreateLicenseCheckLog(ctx context.Context, log *models.LicenseCheckLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockLogStore) CreateAdminLog(ctx context.Context, log *models.AdminLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockLogStore) GetLicenseCheckLogsByLicenseKey(ctx context.Context, licenseKey string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error) {
	args := m.Called(ctx, licenseKey, statusCode, pagination)
	return args.Get(0).([]models.LicenseCheckLog), args.Int(1), args.Error(2)
}

func (m *MockLogStore) GetLicenseCheckLogsByProductID(ctx context.Context, productID string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error) {
	args := m.Called(ctx, productID, statusCode, pagination)
	return args.Get(0).([]models.LicenseCheckLog), args.Int(1), args.Error(2)
}

func (m *MockLogStore) GetLicenseCheckLogsByProductGroupID(ctx context.Context, productGroupID string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error) {
	args := m.Called(ctx, productGroupID, statusCode, pagination)
	return args.Get(0).([]models.LicenseCheckLog), args.Int(1), args.Error(2)
}

func (m *MockLogStore) ListAdminLogs(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.AdminLog, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]models.AdminLog), args.Int(1), args.Error(2)
}

func TestCreateProductHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)
	// Allow async logging
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	router := gin.New()
	router.POST("/admin/products", handlers.CreateProductHandler(mockProductStore, mockLogStore))

	t.Run("Success", func(t *testing.T) {
		type createProductRequest struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		reqBody := createProductRequest{
			Name:        "Test Product",
			Description: "A test product",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("CreateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
			return p.Name == reqBody.Name && p.Description == reqBody.Description
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestReleaseHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockReleaseStore := new(MockReleaseStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	router := gin.New()
	
	// Register consolidated handlers
	router.POST("/admin/releases", handlers.CreateReleaseHandler(mockReleaseStore, mockLogStore))
	router.GET("/admin/releases/global", handlers.ListGlobalReleasesHandler(mockReleaseStore))
	router.GET("/admin/releases", handlers.ListAllReleasesHandler(mockReleaseStore))
	router.PUT("/admin/releases/:releaseId", handlers.UpdateReleaseHandler(mockReleaseStore, mockLogStore))

	t.Run("CreateRelease_Success", func(t *testing.T) {
		productID := uuid.New()
		type createReleaseRequest struct {
			Version   string  `json:"version"`
			ProductID *string `json:"product_id"`
		}
		pidStr := productID.String()
		reqBody := createReleaseRequest{
			Version:   "1.0.0",
			ProductID: &pidStr,
		}
		body, _ := json.Marshal(reqBody)

		mockReleaseStore.On("CreateRelease", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
			return r.Version == reqBody.Version && r.ProductID != nil && *r.ProductID == productID
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/releases", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})

	t.Run("CreateGlobalRelease_Success", func(t *testing.T) {
		type createReleaseRequest struct {
			Version string `json:"version"`
		}
		reqBody := createReleaseRequest{
			Version: "1.0.0-global",
		}
		body, _ := json.Marshal(reqBody)

		mockReleaseStore.On("CreateRelease", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
			return r.Version == reqBody.Version && r.ProductID == nil && r.ProductGroupID == nil
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/releases", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})

	t.Run("ListGlobalReleases_Success", func(t *testing.T) {
		expectedReleases := []models.Release{
			{ID: uuid.New(), Version: "1.0.0-global", ProductID: nil},
		}

		mockReleaseStore.On("ListGlobalReleases", mock.Anything, mock.MatchedBy(func(ownerId *string) bool {
			return ownerId == nil
		}), mock.Anything).Return(expectedReleases, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/releases/global", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})

	t.Run("ListAllReleases_Success", func(t *testing.T) {
		expectedReleases := []models.Release{
			{ID: uuid.New(), Version: "1.0.0-global", ProductID: nil},
			{ID: uuid.New(), Version: "1.0.1-product", ProductID: &uuid.UUID{}},
		}

		mockReleaseStore.On("ListAllReleases", mock.Anything, mock.MatchedBy(func(ownerId *string) bool {
			return ownerId == nil
		}), mock.Anything).Return(expectedReleases, 2, nil)

		req, _ := http.NewRequest("GET", "/admin/releases", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})

	t.Run("UpdateRelease_Success", func(t *testing.T) {
		releaseID := uuid.New()
		reqBody := map[string]interface{}{
			"version": "1.0.1-updated",
		}
		body, _ := json.Marshal(reqBody)


		mockReleaseStore.On("GetRelease", mock.Anything, releaseID.String()).Return(&models.Release{ID: releaseID, OwnerID: nil}, nil)
		mockReleaseStore.On("UpdateRelease", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
			return r.ID == releaseID && r.Version == "1.0.1-updated"
		})).Return(nil)

		req, _ := http.NewRequest("PUT", "/admin/releases/"+releaseID.String(), bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})
}

func TestFeatureHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockFeatureStore := new(MockFeatureStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router.POST("/admin/features", handlers.CreateFeatureHandler(mockFeatureStore, mockLogStore))
	router.GET("/admin/features/global", handlers.ListGlobalFeaturesHandler(mockFeatureStore))
	router.GET("/admin/features", handlers.ListAllFeaturesHandler(mockFeatureStore))
	router.PUT("/admin/features/:featureId", handlers.UpdateFeatureHandler(mockFeatureStore, mockLogStore))

	type createFeatureRequest struct {
		Name      string  `json:"name"`
		Code      string  `json:"code"`
		ProductID *string `json:"product_id"`
	}

	t.Run("CreateFeature_Success", func(t *testing.T) {
		productID := uuid.New()
		pidStr := productID.String()
		reqBody := createFeatureRequest{
			Name:      "Test Feature",
			Code:      "TEST_FEAT",
			ProductID: &pidStr,
		}
		body, _ := json.Marshal(reqBody)

		mockFeatureStore.On("CreateFeature", mock.Anything, mock.MatchedBy(func(f *models.Feature) bool {
			return f.ProductID != nil && *f.ProductID == productID && f.Name == reqBody.Name
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/features", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})

	t.Run("CreateGlobalFeature_Success", func(t *testing.T) {
		reqBody := createFeatureRequest{
			Name: "Global Feature",
			Code: "GLOBAL_FEAT",
		}
		body, _ := json.Marshal(reqBody)

		mockFeatureStore.On("CreateFeature", mock.Anything, mock.MatchedBy(func(f *models.Feature) bool {
			return f.Name == reqBody.Name && f.Code == reqBody.Code && f.ProductID == nil && f.ProductGroupID == nil
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/features", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})

	t.Run("ListGlobalFeatures_Success", func(t *testing.T) {
		expectedFeatures := []models.Feature{
			{ID: uuid.New(), Name: "Global Feature", Code: "GLOBAL", ProductID: nil},
		}

		mockFeatureStore.On("ListGlobalFeatures", mock.Anything, mock.MatchedBy(func(ownerId *string) bool {
			return ownerId == nil
		}), mock.Anything).Return(expectedFeatures, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/features/global", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})

	t.Run("ListAllFeatures_Success", func(t *testing.T) {
		expectedFeatures := []models.Feature{
			{ID: uuid.New(), Name: "Global Feature", Code: "GLOBAL", ProductID: nil},
			{ID: uuid.New(), Name: "Product Feature", Code: "PROD", ProductID: &uuid.UUID{}},
		}

		mockFeatureStore.On("ListAllFeatures", mock.Anything, mock.MatchedBy(func(ownerId *string) bool {
			return ownerId == nil
		}), mock.Anything).Return(expectedFeatures, 2, nil)

		req, _ := http.NewRequest("GET", "/admin/features", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})

	t.Run("UpdateFeature_Success", func(t *testing.T) {
		featureID := uuid.New()
		reqBody := map[string]interface{}{
			"name":        "Updated Feature",
			"code":        "UPDATED",
			"description": "New desc",
		}
		body, _ := json.Marshal(reqBody)

		mockFeatureStore.On("GetFeature", mock.Anything, featureID.String()).Return(&models.Feature{ID: featureID, OwnerID: nil}, nil)
		mockFeatureStore.On("UpdateFeature", mock.Anything, mock.MatchedBy(func(f *models.Feature) bool {
			return f.ID == featureID && f.Name == "Updated Feature" && f.Code == "UPDATED"
		})).Return(nil)

		req, _ := http.NewRequest("PUT", "/admin/features/"+featureID.String(), bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})
}

func TestUpdateLicenseHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	router.PUT("/admin/keys", handlers.UpdateLicenseHandler(mockLicenseStore, mockLogStore))

	t.Run("UpdateLicense_Success", func(t *testing.T) {
		key := "test-key"
		existingLicense := &models.License{
			ID:        uuid.New(),
			Key:       key,
			Type:      models.LicenseTypePerpetual,
			ProductID: uuid.New(),
		}

		type updateLicenseRequest struct {
			Type         models.LicenseType `json:"type"`
			FeatureCodes []string           `json:"feature_codes"`
		}
		reqBody := updateLicenseRequest{
			Type:       models.LicenseTypeTimed,
			FeatureCodes: []string{"sso"},
		}
		body, _ := json.Marshal(reqBody)

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(existingLicense, nil)
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.Type == reqBody.Type && len(l.Features) == 1 && l.Features[0] == "sso"
		})).Return(nil)
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(existingLicense, nil)

		req, _ := http.NewRequest("PUT", "/admin/keys", bytes.NewBuffer(body))
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLicenseStore.AssertExpectations(t)
	})
}

func TestGetLicenseHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	router := gin.New()
	router.GET("/admin/keys", handlers.GetLicenseHandler(mockLicenseStore))

	t.Run("GetLicense_ByKey_Success", func(t *testing.T) {
		key := "test-key"
		license := &models.License{
			ID:  uuid.New(),
			Key: key,
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/admin/keys", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("GetLicense_All_Success", func(t *testing.T) {
		licenses := []models.License{
			{ID: uuid.New(), Key: "key1"},
			{ID: uuid.New(), Key: "key2"},
		}
		mockLicenseStore.On("ListLicenses", mock.Anything, mock.Anything, mock.Anything).Return(licenses, 2, nil).Once()

		req, _ := http.NewRequest("GET", "/admin/keys", nil)
		// No X-License-Key header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp models.PaginatedList[models.License]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		mockLicenseStore.AssertExpectations(t)
	})
}


func TestProductCRUDHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	router.POST("/admin/products", handlers.CreateProductHandler(mockProductStore, mockLogStore))
	router.GET("/admin/products/:id", handlers.GetProductHandler(mockProductStore))
	router.PUT("/admin/products/:id", handlers.UpdateProductHandler(mockProductStore, mockLogStore))
	router.DELETE("/admin/products/:id", handlers.DeleteProductHandler(mockProductStore, mockLogStore))

	t.Run("GetProduct_Success", func(t *testing.T) {
		id := uuid.New().String()
		product := &models.Product{
			ID:   uuid.MustParse(id),
			Name: "Existing Product",
		}
		mockProductStore.On("GetProduct", mock.Anything, id).Return(product, nil)

		req, _ := http.NewRequest("GET", "/admin/products/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockProductStore.AssertExpectations(t)
	})

	t.Run("UpdateProduct_Success", func(t *testing.T) {
		id := uuid.New().String()
		existingProduct := &models.Product{
			ID:   uuid.MustParse(id),
			Name: "Old Name",
		}
		type updateProductRequest struct {
			Name string `json:"name"`
		}
		reqBody := updateProductRequest{
			Name: "New Name",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, id).Return(existingProduct, nil)
		mockProductStore.On("UpdateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
			return p.Name == "New Name" && p.ID.String() == id
		})).Return(nil)

		req, _ := http.NewRequest("PUT", "/admin/products/"+id, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockProductStore.AssertExpectations(t)
	})

	t.Run("UpdateProduct_LicenseFields_Success", func(t *testing.T) {
		id := uuid.New().String()
		existingProduct := &models.Product{
			ID:   uuid.MustParse(id),
			Name: "Product",
		}
		type updateProductRequest struct {
			LicenseType     models.LicenseType `json:"license_type"`
			LicenseDuration string             `json:"license_duration"`
		}
		reqBody := updateProductRequest{
			LicenseType:     models.LicenseTypeTimed,
			LicenseDuration: "30d",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, id).Return(existingProduct, nil)
		mockProductStore.On("UpdateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
			return p.LicenseType == models.LicenseTypeTimed && p.LicenseDuration == "30d" && p.ID.String() == id
		})).Return(nil)

		req, _ := http.NewRequest("PUT", "/admin/products/"+id, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockProductStore.AssertExpectations(t)
	})

	t.Run("DeleteProduct_Success", func(t *testing.T) {
		id := uuid.New().String()
		mockProductStore.On("GetProduct", mock.Anything, id).Return(&models.Product{ID: uuid.MustParse(id), OwnerID: nil}, nil)
		mockProductStore.On("DeleteProduct", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest("DELETE", "/admin/products/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockProductStore.AssertExpectations(t)
	})

	t.Run("CreateProduct_WithCustomSeparator_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":           "Separator Product",
			"license_prefix": "SEP",
			"license_separator":  "_",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("CreateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
			return p.Name == "Separator Product" && p.LicenseSeparator == "_"
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockProductStore.AssertExpectations(t)
	})
}

func TestGenerateLicenseWithCustomSeparator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	router := gin.New()
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	mockProductGroupStore := new(MockProductGroupStore)
	router.POST("/admin/keys", handlers.GenerateLicenseHandler(mockLicenseStore, mockProductStore, mockProductGroupStore, mockLogStore))

	t.Run("Success_CustomSeparator", func(t *testing.T) {
		productID := uuid.New()
		product := &models.Product{
			ID:            productID,
			Name:          "Custom Sep Product",
			LicensePrefix:    "CUSTOM",
			LicenseSeparator: "#",
		}

		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       models.LicenseTypePerpetual,
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil)
		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.ProductID == productID && strings.HasPrefix(l.Key, "CUSTOM#")
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockProductStore.AssertExpectations(t)
		mockLicenseStore.AssertExpectations(t)
	})
}


func TestCheckLicenseHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)

	mockLogStore.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	
	router := gin.New()
	router.GET("/check", handlers.CheckLicenseHandler(mockLicenseStore, mockProductStore, "", mockLogStore))

	t.Run("ValidLicense_NoRestrictions", func(t *testing.T) {
		key := "TEST-key123"
		license := &models.License{
			ID:        uuid.New(),
			Key:       key,
			Type:      models.LicenseTypePerpetual,
			Features:  []string{},
			Releases:  []string{},
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["valid"].(bool))
		assert.Nil(t, resp["reason"])
	})

	t.Run("VersionValidation_Allowed", func(t *testing.T) {
		key := "TEST-ver123"
		license := &models.License{
			ID:   uuid.New(),
			Key:  key,
			Type: models.LicenseTypePerpetual,
			Releases: []string{"1.0.0", "2.0.0"},
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check?version=1.0.0", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["valid"].(bool))
	})

	t.Run("VersionValidation_NotAllowed", func(t *testing.T) {
		key := "TEST-verblocked"
		license := &models.License{
			ID:   uuid.New(),
			Key:  key,
			Type: models.LicenseTypePerpetual,
			Releases: []string{"1.0.0"},
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check?version=3.0.0", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp["valid"].(bool))
		assert.Equal(t, "License not valid for version 3.0.0", resp["reason"])
	})

	t.Run("VersionValidation_NoRestrictions_AllVersionsAllowed", func(t *testing.T) {
		key := "TEST-norestrict"
		license := &models.License{
			ID:       uuid.New(),
			Key:      key,
			Type:     models.LicenseTypePerpetual,
			Releases: []string{},
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check?version=any-version", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["valid"].(bool))
	})

	t.Run("FeatureValidation_Allowed", func(t *testing.T) {
		key := "TEST-feat123"
		license := &models.License{
			ID:   uuid.New(),
			Key:  key,
			Type: models.LicenseTypePerpetual,
			Features: []string{"sso", "premium"},
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check?feature=sso", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["valid"].(bool))
	})

	t.Run("FeatureValidation_NotAllowed", func(t *testing.T) {
		key := "TEST-featblocked"
		license := &models.License{
			ID:   uuid.New(),
			Key:  key,
			Type: models.LicenseTypePerpetual,
			Features: []string{"basic"},
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check?feature=enterprise", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp["valid"].(bool))
		assert.Equal(t, "Feature not enabled: enterprise", resp["reason"])
	})

	t.Run("ExpiredLicense", func(t *testing.T) {
		key := "TEST-expired"
		pastTime := time.Now().Add(-24 * time.Hour)
		license := &models.License{
			ID:        uuid.New(),
			Key:       key,
			Type:      models.LicenseTypeTimed,
			ExpiresAt: &pastTime,
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp["valid"].(bool))
		assert.Equal(t, "License has expired", resp["reason"])
	})
}

func TestGenerateLicenseWithLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	mockProductGroupStore := new(MockProductGroupStore)
	router := gin.New()
	router.POST("/admin/keys", handlers.GenerateLicenseHandler(mockLicenseStore, mockProductStore, mockProductGroupStore, mockLogStore))

	t.Run("LengthFromRequest", func(t *testing.T) {
		productID := uuid.New()
		product := &models.Product{
			ID: productID,
		}
		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       models.LicenseTypePerpetual,
			"length":     10,
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil)
		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			parts := strings.Split(l.Key, "-")
			return len(parts) == 2 && len(parts[1]) == 10
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("LengthFromProduct", func(t *testing.T) {
		productID := uuid.New()
		product := &models.Product{
			ID:            productID,
			LicenseLength: 15,
		}
		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       models.LicenseTypePerpetual,
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil)
		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			parts := strings.Split(l.Key, "-")
			return len(parts) == 2 && len(parts[1]) == 15
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("LengthFromGroup", func(t *testing.T) {
		productID := uuid.New()
		groupID := uuid.New()
		product := &models.Product{
			ID:             productID,
			ProductGroupID: &groupID,
		}
		group := &models.ProductGroup{
			ID:            groupID,
			LicenseLength: 8,
		}
		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       models.LicenseTypePerpetual,
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil)
		mockProductGroupStore.On("GetProductGroup", mock.Anything, groupID.String()).Return(group, nil)
		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			parts := strings.Split(l.Key, "-")
			return len(parts) == 2 && len(parts[1]) == 8
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("DefaultLength", func(t *testing.T) {
		productID := uuid.New()
		product := &models.Product{
			ID: productID,
		}
		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       models.LicenseTypePerpetual,
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil)
		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			parts := strings.Split(l.Key, "-")
			return len(parts) == 2 && len(parts[1]) == 12 // Default 12
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestLogHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLogStore := new(MockLogStore)
	router := gin.New()
	router.GET("/admin/logs/license-checks", handlers.GetLicenseCheckLogsHandler(mockLogStore))
	router.GET("/admin/logs/admin-actions", handlers.GetAdminLogsHandler(mockLogStore))

	t.Run("GetLicenseCheckLogsByLicenseKey", func(t *testing.T) {
		key := "TEST-KEY"
		logs := []models.LicenseCheckLog{
			{ID: uuid.New(), LicenseKey: key},
		}
		mockLogStore.On("GetLicenseCheckLogsByLicenseKey", mock.Anything, key, mock.Anything, mock.Anything).Return(logs, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?license_key="+key, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLogStore.AssertExpectations(t)
	})

	t.Run("GetLicenseCheckLogsByProductID", func(t *testing.T) {
		id := uuid.New().String()
		logs := []models.LicenseCheckLog{
			{ID: uuid.New(), ProductID: &uuid.Nil},
		}
		mockLogStore.On("GetLicenseCheckLogsByProductID", mock.Anything, id, mock.Anything, mock.Anything).Return(logs, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?product_id="+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLogStore.AssertExpectations(t)
	})

	t.Run("GetLicenseCheckLogsByProductGroupID", func(t *testing.T) {
		id := uuid.New().String()
		logs := []models.LicenseCheckLog{
			{ID: uuid.New()},
		}
		mockLogStore.On("GetLicenseCheckLogsByProductGroupID", mock.Anything, id, mock.Anything, mock.Anything).Return(logs, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?product_group_id="+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLogStore.AssertExpectations(t)
	})

	t.Run("GetAdminLogsByOwnerID", func(t *testing.T) {
		ownerID := "test-owner-id"
		logs := []models.AdminLog{
			{ID: uuid.New(), OwnerID: &ownerID},
		}
		// Expect ListAdminLogs with specific ownerID
		mockLogStore.On("ListAdminLogs", mock.Anything, &ownerID, mock.Anything).Return(logs, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/logs/admin-actions?owner_id="+ownerID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLogStore.AssertExpectations(t)
	})
}

func TestProductGroupFeatureReleaseHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockFeatureStore := new(MockFeatureStore)
	mockReleaseStore := new(MockReleaseStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	
	// Register consolidated handlers
	router.POST("/admin/features", handlers.CreateFeatureHandler(mockFeatureStore, mockLogStore))
	router.GET("/admin/features", handlers.ListAllFeaturesHandler(mockFeatureStore))
	router.POST("/admin/releases", handlers.CreateReleaseHandler(mockReleaseStore, mockLogStore))
	router.GET("/admin/releases", handlers.ListAllReleasesHandler(mockReleaseStore))

	groupID := uuid.New().String()

	t.Run("CreateProductGroupFeature_Success", func(t *testing.T) {
		type createFeatureRequest struct {
			Name           string  `json:"name"`
			Code           string  `json:"code"`
			ProductGroupID *string `json:"product_group_id"`
		}
		reqBody := createFeatureRequest{
			Name:           "Group Feature",
			Code:           "GRP_FEAT",
			ProductGroupID: &groupID,
		}
		body, _ := json.Marshal(reqBody)

		mockFeatureStore.On("CreateFeature", mock.Anything, mock.MatchedBy(func(f *models.Feature) bool {
			return f.Name == reqBody.Name && f.Code == reqBody.Code && f.ProductGroupID != nil && f.ProductGroupID.String() == groupID
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/features", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})

	t.Run("CreateProductGroupRelease_Success", func(t *testing.T) {
		type createReleaseRequest struct {
			Version        string  `json:"version"`
			ProductGroupID *string `json:"product_group_id"`
		}
		reqBody := createReleaseRequest{
			Version:        "v1.0-group",
			ProductGroupID: &groupID,
		}
		body, _ := json.Marshal(reqBody)

		mockReleaseStore.On("CreateRelease", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
			return r.Version == reqBody.Version && r.ProductGroupID != nil && r.ProductGroupID.String() == groupID
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/releases", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})

	t.Run("ListProductGroupFeatures_Success", func(t *testing.T) {
		expectedFeatures := []models.Feature{
			{ID: uuid.New(), Name: "Group Feature", Code: "GRP_FEAT", ProductGroupID: &uuid.UUID{}},
		}

		mockFeatureStore.On("ListFeaturesByProductGroup", mock.Anything, groupID, mock.Anything, mock.Anything).Return(expectedFeatures, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/features?product_group_id="+groupID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockFeatureStore.AssertExpectations(t)
	})

	t.Run("ListProductGroupReleases_Success", func(t *testing.T) {
		expectedReleases := []models.Release{
			{ID: uuid.New(), Version: "v1.0-group", ProductGroupID: &uuid.UUID{}},
		}

		mockReleaseStore.On("ListReleasesByProductGroup", mock.Anything, groupID, mock.Anything, mock.Anything).Return(expectedReleases, 1, nil)

		req, _ := http.NewRequest("GET", "/admin/releases?product_group_id="+groupID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockReleaseStore.AssertExpectations(t)
	})
}

