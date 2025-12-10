package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/array/banking-api/internal/services"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// HealthCheckHandler handles the health check endpoint
type HealthCheckHandler struct {
	db              *gorm.DB
	northwindClient services.NorthwindClientInterface
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(db *gorm.DB, northwindClient services.NorthwindClientInterface) *HealthCheckHandler {
	return &HealthCheckHandler{
		db:              db,
		northwindClient: northwindClient,
	}
}

// HealthCheck adds the health check endpoint
// @Summary Health check
// @Description Check the health of the API and its dependencies (e.g., database, Northwind API)
// @Tags Health
// @Produce json
// @Success 200 {object} object{status=string,timestamp=string,dependencies=object{database=string,northwind_api=string}} "API is healthy"
// @Failure 503 {object} object{status=string,timestamp=string,dependencies=object{database=string,northwind_api=string}} "API is unhealthy"
// @Router /health [get]
func (h *HealthCheckHandler) HealthCheck(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	isHealthy := true
	dependencies := make(map[string]string)

	// Check database
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		dependencies["database"] = "error"
		isHealthy = false
	} else {
		dependencies["database"] = "ok"
	}

	// Check Northwind API
	if err := h.northwindClient.HealthCheck(ctx); err != nil {
		dependencies["northwind_api"] = "error"
		isHealthy = false
		c.Logger().Errorf("Northwind API health check failed: %v", err)
	} else {
		dependencies["northwind_api"] = "ok"
	}

	status := "ok"
	statusCode := http.StatusOK
	if !isHealthy {
		status = "error"
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, map[string]interface{}{
		"status":       status,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"dependencies": dependencies,
	})
}
