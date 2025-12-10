package storefront

import (
	"net/http"
	"time"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
)

// HomeHandler handles the storefront homepage
type HomeHandler struct {
	productService domain.ProductService
	renderer       *handler.Renderer
}

// NewHomeHandler creates a new home handler
func NewHomeHandler(productService domain.ProductService, renderer *handler.Renderer) *HomeHandler {
	return &HomeHandler{
		productService: productService,
		renderer:       renderer,
	}
}

// HomePageData contains data for the home page template
type HomePageData struct {
	StoreName        string
	FeaturedProducts []FeaturedProduct
	Year             int
	User             interface{}
	CartCount        int
	CSRFToken        string
}

// FeaturedProduct is a simplified product for the home page
type FeaturedProduct struct {
	Slug          string
	Name          string
	Origin        string
	ImageURL      string
	StartingPrice int32 // cents
}

// ServeHTTP handles GET /
func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get featured products (take first 3)
	products, err := h.productService.ListProducts(ctx)
	featured := make([]FeaturedProduct, 0, 3)
	if err == nil {
		for i, p := range products {
			if i >= 3 {
				break
			}
			fp := FeaturedProduct{
				Slug: p.Slug,
				Name: p.Name,
			}
			if p.Origin.Valid {
				fp.Origin = p.Origin.String
			}
			if p.PrimaryImageURL.Valid {
				fp.ImageURL = p.PrimaryImageURL.String
			}
			// Note: Price would require loading SKUs - skip for home page simplicity
			featured = append(featured, fp)
		}
	}

	data := HomePageData{
		StoreName:        "Freyja Coffee", // TODO: Get from tenant config
		FeaturedProducts: featured,
		Year:             time.Now().Year(),
		User:             middleware.GetUserFromContext(ctx),
		CSRFToken:        middleware.GetCSRFToken(ctx),
	}

	// Get cart count if available
	if cartCount, ok := ctx.Value("cart_count").(int); ok {
		data.CartCount = cartCount
	}

	h.renderer.RenderHTTP(w, "storefront/home", data)
}
