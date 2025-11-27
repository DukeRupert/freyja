-- +goose Up
-- +goose StatementBegin

-- Product categories: hierarchical product organization
CREATE TABLE product_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Category details
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,

    -- Hierarchy (self-referencing for parent/child)
    parent_id UUID REFERENCES product_categories(id) ON DELETE CASCADE,

    -- Display
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- SEO
    meta_title VARCHAR(255),
    meta_description TEXT,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT product_categories_tenant_slug_unique UNIQUE (tenant_id, slug)
);

-- Product category assignments
CREATE TABLE product_category_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES product_categories(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT product_category_assignments_unique UNIQUE (product_id, category_id)
);

-- Product tags: flexible product labeling
CREATE TABLE product_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Tag details
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT product_tags_tenant_slug_unique UNIQUE (tenant_id, slug)
);

-- Product tag assignments
CREATE TABLE product_tag_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES product_tags(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT product_tag_assignments_unique UNIQUE (product_id, tag_id)
);

-- Product reviews: customer product reviews
CREATE TABLE product_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL, -- Optional: link to verified purchase

    -- Review content
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title VARCHAR(255),
    review_text TEXT,

    -- Moderation
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'flagged')),
    moderated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    moderated_at TIMESTAMP WITH TIME ZONE,
    moderation_notes TEXT,

    -- Helpfulness tracking
    helpful_count INTEGER NOT NULL DEFAULT 0,
    not_helpful_count INTEGER NOT NULL DEFAULT 0,

    -- Verified purchase
    is_verified_purchase BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- One review per user per product
    CONSTRAINT product_reviews_user_product_unique UNIQUE (user_id, product_id)
);

-- Review helpfulness tracking
CREATE TABLE review_helpfulness (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    review_id UUID NOT NULL REFERENCES product_reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Was this review helpful?
    is_helpful BOOLEAN NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- One vote per user per review
    CONSTRAINT review_helpfulness_unique UNIQUE (review_id, user_id)
);

-- Discount codes: promotional codes
CREATE TABLE discount_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Code details
    code VARCHAR(100) NOT NULL,
    description TEXT,

    -- Discount type
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount', 'free_shipping')),
    discount_value INTEGER NOT NULL, -- Percentage (0-100) or cents

    -- Applicability
    applies_to VARCHAR(20) NOT NULL DEFAULT 'order' CHECK (applies_to IN ('order', 'product', 'category')),
    minimum_order_cents INTEGER,

    -- Usage limits
    usage_limit INTEGER, -- NULL = unlimited
    usage_count INTEGER NOT NULL DEFAULT 0,
    usage_limit_per_customer INTEGER,

    -- Validity
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT discount_codes_tenant_code_unique UNIQUE (tenant_id, code)
);

-- Discount code usage tracking
CREATE TABLE discount_code_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    discount_code_id UUID NOT NULL REFERENCES discount_codes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,

    -- Discount applied
    discount_amount_cents INTEGER NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Email templates: customizable transactional emails
CREATE TABLE email_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Template identification
    template_type VARCHAR(100) NOT NULL, -- e.g., 'order_confirmation', 'shipping_confirmation'
    name VARCHAR(255) NOT NULL,

    -- Email content
    subject VARCHAR(255) NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,

    -- Settings
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT email_templates_tenant_type_unique UNIQUE (tenant_id, template_type)
);

-- Indexes
CREATE INDEX idx_product_categories_tenant_id ON product_categories(tenant_id);
CREATE INDEX idx_product_categories_parent_id ON product_categories(parent_id);
CREATE INDEX idx_product_categories_active ON product_categories(tenant_id, is_active) WHERE is_active = TRUE;

CREATE INDEX idx_product_category_assignments_tenant_id ON product_category_assignments(tenant_id);
CREATE INDEX idx_product_category_assignments_product_id ON product_category_assignments(product_id);
CREATE INDEX idx_product_category_assignments_category_id ON product_category_assignments(category_id);

CREATE INDEX idx_product_tags_tenant_id ON product_tags(tenant_id);

CREATE INDEX idx_product_tag_assignments_tenant_id ON product_tag_assignments(tenant_id);
CREATE INDEX idx_product_tag_assignments_product_id ON product_tag_assignments(product_id);
CREATE INDEX idx_product_tag_assignments_tag_id ON product_tag_assignments(tag_id);

CREATE INDEX idx_product_reviews_tenant_id ON product_reviews(tenant_id);
CREATE INDEX idx_product_reviews_product_id ON product_reviews(product_id);
CREATE INDEX idx_product_reviews_user_id ON product_reviews(user_id);
CREATE INDEX idx_product_reviews_status ON product_reviews(tenant_id, status);
CREATE INDEX idx_product_reviews_approved ON product_reviews(product_id, status)
    WHERE status = 'approved';

CREATE INDEX idx_review_helpfulness_tenant_id ON review_helpfulness(tenant_id);
CREATE INDEX idx_review_helpfulness_review_id ON review_helpfulness(review_id);
CREATE INDEX idx_review_helpfulness_user_id ON review_helpfulness(user_id);

CREATE INDEX idx_discount_codes_tenant_id ON discount_codes(tenant_id);
CREATE INDEX idx_discount_codes_code ON discount_codes(tenant_id, code);
CREATE INDEX idx_discount_codes_active ON discount_codes(tenant_id, is_active, starts_at, expires_at)
    WHERE is_active = TRUE;

CREATE INDEX idx_discount_code_usage_tenant_id ON discount_code_usage(tenant_id);
CREATE INDEX idx_discount_code_usage_code_id ON discount_code_usage(discount_code_id);
CREATE INDEX idx_discount_code_usage_user_id ON discount_code_usage(user_id);
CREATE INDEX idx_discount_code_usage_order_id ON discount_code_usage(order_id);

CREATE INDEX idx_email_templates_tenant_id ON email_templates(tenant_id);
CREATE INDEX idx_email_templates_type ON email_templates(tenant_id, template_type);
CREATE INDEX idx_email_templates_active ON email_templates(tenant_id, is_active) WHERE is_active = TRUE;

-- Auto-update triggers
CREATE TRIGGER update_product_categories_updated_at
    BEFORE UPDATE ON product_categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_product_reviews_updated_at
    BEFORE UPDATE ON product_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_discount_codes_updated_at
    BEFORE UPDATE ON discount_codes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_email_templates_updated_at
    BEFORE UPDATE ON email_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update review helpfulness counts
CREATE OR REPLACE FUNCTION update_review_helpfulness_counts()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE product_reviews
    SET
        helpful_count = (
            SELECT COUNT(*)
            FROM review_helpfulness
            WHERE review_id = COALESCE(NEW.review_id, OLD.review_id)
            AND is_helpful = TRUE
        ),
        not_helpful_count = (
            SELECT COUNT(*)
            FROM review_helpfulness
            WHERE review_id = COALESCE(NEW.review_id, OLD.review_id)
            AND is_helpful = FALSE
        )
    WHERE id = COALESCE(NEW.review_id, OLD.review_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_review_counts_on_helpfulness
    AFTER INSERT OR UPDATE OR DELETE ON review_helpfulness
    FOR EACH ROW
    EXECUTE FUNCTION update_review_helpfulness_counts();

-- Trigger to increment discount code usage count
CREATE OR REPLACE FUNCTION increment_discount_code_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE discount_codes
    SET usage_count = usage_count + 1
    WHERE id = NEW.discount_code_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER increment_usage_on_discount_application
    AFTER INSERT ON discount_code_usage
    FOR EACH ROW
    EXECUTE FUNCTION increment_discount_code_usage();

COMMENT ON TABLE product_categories IS 'Hierarchical product categories';
COMMENT ON TABLE product_tags IS 'Flexible product tags';
COMMENT ON TABLE product_reviews IS 'Customer product reviews with moderation';
COMMENT ON TABLE review_helpfulness IS 'User votes on review helpfulness';
COMMENT ON TABLE discount_codes IS 'Promotional discount codes';
COMMENT ON TABLE discount_code_usage IS 'Tracking of discount code redemptions';
COMMENT ON TABLE email_templates IS 'Customizable transactional email templates';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS increment_usage_on_discount_application ON discount_code_usage;
DROP FUNCTION IF EXISTS increment_discount_code_usage();
DROP TRIGGER IF EXISTS update_review_counts_on_helpfulness ON review_helpfulness;
DROP FUNCTION IF EXISTS update_review_helpfulness_counts();
DROP TRIGGER IF EXISTS update_email_templates_updated_at ON email_templates;
DROP TRIGGER IF EXISTS update_discount_codes_updated_at ON discount_codes;
DROP TRIGGER IF EXISTS update_product_reviews_updated_at ON product_reviews;
DROP TRIGGER IF EXISTS update_product_categories_updated_at ON product_categories;
DROP TABLE IF EXISTS email_templates CASCADE;
DROP TABLE IF EXISTS discount_code_usage CASCADE;
DROP TABLE IF EXISTS discount_codes CASCADE;
DROP TABLE IF EXISTS review_helpfulness CASCADE;
DROP TABLE IF EXISTS product_reviews CASCADE;
DROP TABLE IF EXISTS product_tag_assignments CASCADE;
DROP TABLE IF EXISTS product_tags CASCADE;
DROP TABLE IF EXISTS product_category_assignments CASCADE;
DROP TABLE IF EXISTS product_categories CASCADE;
-- +goose StatementEnd
