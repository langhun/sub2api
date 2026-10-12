package xznoauthinvoice

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

type Module struct {
	Handler *Handler
}

func NewModule(db *sql.DB, cfg config.XZNOAuthInvoiceConfig) *Module {
	return &Module{Handler: NewHandler(NewClient(cfg), NewRepository(db))}
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	if m == nil || m.Handler == nil || r == nil {
		return
	}
	invoices := r.Group("/custom/invoices")
	invoices.POST("/validate", m.Handler.Validate)
	invoices.POST("/eligible-orders", m.Handler.EligibleOrders)
	invoices.POST("", m.Handler.Create)
	invoices.GET("", m.Handler.List)
	invoices.POST("/tax-payments/status", m.Handler.TaxStatus)
	invoices.POST("/:id/tax-payments", m.Handler.Checkout)
	invoices.POST("/:id/cancel", m.Handler.Cancel)
	invoices.GET("/:id/pdf", m.Handler.PDF)
}
