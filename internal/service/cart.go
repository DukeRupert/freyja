package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// CartService provides business logic for shopping cart operations
type CartService interface {
	GetOrCreateCart(ctx context.Context, sessionID string) (*Cart, string, error)
	GetCart(ctx context.Context, sessionID string) (*Cart, error)
	AddItem(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error)
	UpdateItemQuantity(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error)
	RemoveItem(ctx context.Context, cartID string, skuID string) (*CartSummary, error)
	GetCartSummary(ctx context.Context, cartID string) (*CartSummary, error)
	ClearCart(ctx context.Context, cartID string) error
}

// Cart represents a lightweight cart view model
type Cart struct {
	ID        pgtype.UUID
	TenantID  pgtype.UUID
	SessionID pgtype.UUID
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

// CartSummary aggregates cart information with items and calculated totals
type CartSummary struct {
	Cart      Cart
	Items     []CartItem
	Subtotal  int32
	ItemCount int
}

// CartItem represents a cart line item with product details and calculated totals
type CartItem struct {
	ID             pgtype.UUID
	SKUID          pgtype.UUID
	ProductName    string
	SKU            string
	WeightValue    string
	Grind          string
	Quantity       int32
	UnitPriceCents int32
	LineSubtotal   int32
	ImageURL       string
}

type cartService struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewCartService creates a new CartService instance
func NewCartService(repo repository.Querier, tenantID string) (CartService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &cartService{
		repo:     repo,
		tenantID: tenantUUID,
	}, nil
}

// GetOrCreateCart retrieves an existing cart or creates a new session and cart
// Returns the cart, session ID (new or existing), and any error
func (s *cartService) GetOrCreateCart(ctx context.Context, sessionID string) (*Cart, string, error) {
	var sessionUUID pgtype.UUID

	if sessionID == "" {
		newSessionID, err := GenerateSessionID()
		if err != nil {
			return nil, "", fmt.Errorf("failed to generate session ID: %w", err)
		}
		sessionID = newSessionID

		expiresAt := pgtype.Timestamptz{}
		if err := expiresAt.Scan(time.Now().Add(30 * 24 * time.Hour)); err != nil {
			return nil, "", fmt.Errorf("failed to set session expiration: %w", err)
		}

		session, err := s.repo.CreateSession(ctx, repository.CreateSessionParams{
			Token:     sessionID,
			Data:      []byte("{}"),
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return nil, "", fmt.Errorf("failed to create session: %w", err)
		}
		sessionUUID = session.ID
	} else {
		if err := sessionUUID.Scan(sessionID); err == nil {
			cart, err := s.repo.GetCartBySessionID(ctx, sessionUUID)
			if err == nil {
				return &Cart{
					ID:        cart.ID,
					TenantID:  cart.TenantID,
					SessionID: cart.SessionID,
					CreatedAt: cart.CreatedAt,
					UpdatedAt: cart.UpdatedAt,
				}, sessionID, nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, "", fmt.Errorf("failed to get cart by session ID: %w", err)
			}
		} else {
			session, err := s.repo.GetSessionByToken(ctx, sessionID)
			if err == nil {
				sessionUUID = session.ID

				cart, err := s.repo.GetCartBySessionID(ctx, sessionUUID)
				if err == nil {
					return &Cart{
						ID:        cart.ID,
						TenantID:  cart.TenantID,
						SessionID: cart.SessionID,
						CreatedAt: cart.CreatedAt,
						UpdatedAt: cart.UpdatedAt,
					}, sessionID, nil
				}
				if !errors.Is(err, sql.ErrNoRows) {
					return nil, "", fmt.Errorf("failed to get cart by session ID: %w", err)
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil, "", fmt.Errorf("failed to get session by token: %w", err)
			} else {
				expiresAt := pgtype.Timestamptz{}
				if err := expiresAt.Scan(time.Now().Add(30 * 24 * time.Hour)); err != nil {
					return nil, "", fmt.Errorf("failed to set session expiration: %w", err)
				}

				newSession, err := s.repo.CreateSession(ctx, repository.CreateSessionParams{
					Token:     sessionID,
					Data:      []byte("{}"),
					ExpiresAt: expiresAt,
				})
				if err != nil {
					return nil, "", fmt.Errorf("failed to create session: %w", err)
				}
				sessionUUID = newSession.ID
			}
		}
	}

	cart, err := s.repo.CreateCart(ctx, repository.CreateCartParams{
		TenantID:  s.tenantID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create cart: %w", err)
	}

	return &Cart{
		ID:        cart.ID,
		TenantID:  cart.TenantID,
		SessionID: cart.SessionID,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}, sessionID, nil
}

// GetCart retrieves an existing cart by session ID
func (s *cartService) GetCart(ctx context.Context, sessionID string) (*Cart, error) {
	var sessionUUID pgtype.UUID

	if err := sessionUUID.Scan(sessionID); err == nil {
		cart, err := s.repo.GetCartBySessionID(ctx, sessionUUID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrCartNotFound
			}
			return nil, fmt.Errorf("failed to get cart by session ID: %w", err)
		}

		return &Cart{
			ID:        cart.ID,
			TenantID:  cart.TenantID,
			SessionID: cart.SessionID,
			CreatedAt: cart.CreatedAt,
			UpdatedAt: cart.UpdatedAt,
		}, nil
	}

	session, err := s.repo.GetSessionByToken(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	cart, err := s.repo.GetCartBySessionID(ctx, session.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("failed to get cart by session ID: %w", err)
	}

	return &Cart{
		ID:        cart.ID,
		TenantID:  cart.TenantID,
		SessionID: cart.SessionID,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}, nil
}

// AddItem adds a product SKU to the cart or updates quantity if already present
func (s *cartService) AddItem(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error) {
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	var cartUUID pgtype.UUID
	if err := cartUUID.Scan(cartID); err != nil {
		return nil, fmt.Errorf("invalid cart ID: %w", err)
	}

	var skuUUID pgtype.UUID
	if err := skuUUID.Scan(skuID); err != nil {
		return nil, fmt.Errorf("invalid SKU ID: %w", err)
	}

	sku, err := s.repo.GetSKUByID(ctx, skuUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSKUNotFound
		}
		return nil, fmt.Errorf("failed to get SKU: %w", err)
	}

	priceList, err := s.repo.GetDefaultPriceList(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default price list: %w", err)
	}

	price, err := s.repo.GetPriceForSKU(ctx, repository.GetPriceForSKUParams{
		PriceListID:  priceList.ID,
		ProductSkuID: sku.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPriceNotFound
		}
		return nil, fmt.Errorf("failed to get price for SKU: %w", err)
	}

	_, err = s.repo.AddCartItem(ctx, repository.AddCartItemParams{
		TenantID:       s.tenantID,
		CartID:         cartUUID,
		ProductSkuID:   skuUUID,
		Quantity:       int32(quantity),
		UnitPriceCents: price.PriceCents,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add cart item: %w", err)
	}

	return s.GetCartSummary(ctx, cartID)
}

// UpdateItemQuantity updates the quantity of a cart item
// If quantity is 0, the item is removed
func (s *cartService) UpdateItemQuantity(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error) {
	if quantity == 0 {
		return s.RemoveItem(ctx, cartID, skuID)
	}

	if quantity < 0 {
		return nil, ErrInvalidQuantity
	}

	var cartUUID pgtype.UUID
	if err := cartUUID.Scan(cartID); err != nil {
		return nil, fmt.Errorf("invalid cart ID: %w", err)
	}

	var skuUUID pgtype.UUID
	if err := skuUUID.Scan(skuID); err != nil {
		return nil, fmt.Errorf("invalid SKU ID: %w", err)
	}

	err := s.repo.UpdateCartItemQuantity(ctx, repository.UpdateCartItemQuantityParams{
		CartID:       cartUUID,
		ProductSkuID: skuUUID,
		Quantity:     int32(quantity),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cart item quantity: %w", err)
	}

	return s.GetCartSummary(ctx, cartID)
}

// RemoveItem removes a product SKU from the cart
func (s *cartService) RemoveItem(ctx context.Context, cartID string, skuID string) (*CartSummary, error) {
	var cartUUID pgtype.UUID
	if err := cartUUID.Scan(cartID); err != nil {
		return nil, fmt.Errorf("invalid cart ID: %w", err)
	}

	var skuUUID pgtype.UUID
	if err := skuUUID.Scan(skuID); err != nil {
		return nil, fmt.Errorf("invalid SKU ID: %w", err)
	}

	err := s.repo.RemoveCartItem(ctx, repository.RemoveCartItemParams{
		CartID:       cartUUID,
		ProductSkuID: skuUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to remove cart item: %w", err)
	}

	return s.GetCartSummary(ctx, cartID)
}

// GetCartSummary retrieves a cart with all items and calculated totals
func (s *cartService) GetCartSummary(ctx context.Context, cartID string) (*CartSummary, error) {
	var cartUUID pgtype.UUID
	if err := cartUUID.Scan(cartID); err != nil {
		return nil, fmt.Errorf("invalid cart ID: %w", err)
	}

	cart, err := s.repo.GetCartByID(ctx, cartUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	items, err := s.repo.GetCartItems(ctx, cartUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	cartItems := make([]CartItem, 0, len(items))
	var subtotal int32
	var itemCount int

	for _, item := range items {
		lineSubtotal := item.Quantity * item.UnitPriceCents
		subtotal += lineSubtotal
		itemCount += int(item.Quantity)

		weightValue := ""
		if item.WeightValue.Valid {
			weightValue = fmt.Sprintf("%s%s", item.WeightValue.Int.String(), item.WeightUnit)
		}

		imageURL := ""
		if item.ImageUrl.Valid {
			imageURL = item.ImageUrl.String
		}

		cartItems = append(cartItems, CartItem{
			ID:             item.ID,
			SKUID:          item.ProductSkuID,
			ProductName:    item.ProductName,
			SKU:            item.Sku,
			WeightValue:    weightValue,
			Grind:          item.Grind,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
			LineSubtotal:   lineSubtotal,
			ImageURL:       imageURL,
		})
	}

	return &CartSummary{
		Cart: Cart{
			ID:        cart.ID,
			TenantID:  cart.TenantID,
			SessionID: cart.SessionID,
			CreatedAt: cart.CreatedAt,
			UpdatedAt: cart.UpdatedAt,
		},
		Items:     cartItems,
		Subtotal:  subtotal,
		ItemCount: itemCount,
	}, nil
}

// ClearCart removes all items from a cart
func (s *cartService) ClearCart(ctx context.Context, cartID string) error {
	var cartUUID pgtype.UUID
	if err := cartUUID.Scan(cartID); err != nil {
		return fmt.Errorf("invalid cart ID: %w", err)
	}

	if err := s.repo.ClearCart(ctx, cartUUID); err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	return nil
}
