package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/array/banking-api/internal/services/service_mocks"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type HealthHandlerSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	e               *echo.Echo
	db              *gorm.DB
	northwindClient *service_mocks.MockNorthwindClientInterface
	handler         *HealthCheckHandler
}

func (s *HealthHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.e = echo.New()
	s.e.Validator = NewValidator()

	// Setup in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	s.Require().NoError(err)
	s.db = db

	s.northwindClient = service_mocks.NewMockNorthwindClientInterface(s.ctrl)
	s.handler = NewHealthCheckHandler(s.db, s.northwindClient)
}

func (s *HealthHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func TestHealthHandlerSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerSuite))
}

func (s *HealthHandlerSuite) TestHealthCheck_AllHealthy() {
	s.northwindClient.EXPECT().HealthCheck(gomock.Any()).Return(nil).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.HealthCheck(c)

	s.Require().NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.Require().NoError(err)

	s.Equal("ok", response["status"])
	dependencies := response["dependencies"].(map[string]interface{})
	s.Equal("ok", dependencies["database"])
	s.Equal("ok", dependencies["northwind_api"])
}

func (s *HealthHandlerSuite) TestHealthCheck_DBUnhealthy() {
	sqlDB, _ := s.db.DB()
	sqlDB.Close()

	s.northwindClient.EXPECT().HealthCheck(gomock.Any()).Return(nil).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.HealthCheck(c)

	s.Require().NoError(err)
	s.Equal(http.StatusServiceUnavailable, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.Require().NoError(err)

	s.Equal("error", response["status"])
	dependencies := response["dependencies"].(map[string]interface{})
	s.Equal("error", dependencies["database"])
	s.Equal("ok", dependencies["northwind_api"])
}

func (s *HealthHandlerSuite) TestHealthCheck_NorthwindUnhealthy() {
	s.northwindClient.EXPECT().HealthCheck(gomock.Any()).Return(errors.New("API is down")).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.HealthCheck(c)

	s.Require().NoError(err)
	s.Equal(http.StatusServiceUnavailable, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.Require().NoError(err)

	s.Equal("error", response["status"])
	dependencies := response["dependencies"].(map[string]interface{})
	s.Equal("ok", dependencies["database"])
	s.Equal("error", dependencies["northwind_api"])
}

func (s *HealthHandlerSuite) TestHealthCheck_AllUnhealthy() {
	sqlDB, _ := s.db.DB()
	sqlDB.Close()

	s.northwindClient.EXPECT().HealthCheck(gomock.Any()).Return(errors.New("API is down")).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := s.handler.HealthCheck(c)

	s.Require().NoError(err)
	s.Equal(http.StatusServiceUnavailable, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.Require().NoError(err)

	s.Equal("error", response["status"])
	dependencies := response["dependencies"].(map[string]interface{})
	s.Equal("error", dependencies["database"])
	s.Equal("error", dependencies["northwind_api"])
}
