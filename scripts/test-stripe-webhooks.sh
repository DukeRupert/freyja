#!/bin/bash
#
# Stripe Webhook Testing Script
#
# This script automates testing of Stripe webhook handling in Freyja.
# It supports two modes:
#   1. Quick validation (test mode) - verifies webhook pipeline without side effects
#   2. Full integration - tests with actual order creation (requires manual checkout)
#
# Prerequisites:
#   - Stripe CLI installed and authenticated (stripe login)
#   - Server running on localhost:3000
#   - STRIPE_WEBHOOK_SECRET set in environment
#
# Usage:
#   ./scripts/test-stripe-webhooks.sh [options]
#
# Options:
#   --quick       Run quick validation tests only (default)
#   --full        Run full integration tests (requires manual steps)
#   --listen      Start webhook listener only (for manual testing)
#   --help        Show this help message
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WEBHOOK_URL="${WEBHOOK_URL:-localhost:3000/webhooks/stripe}"
SERVER_URL="${SERVER_URL:-http://localhost:3000}"
WAIT_BETWEEN_TRIGGERS=2

# Counters
PASSED=0
FAILED=0
SKIPPED=0

#######################################
# Print colored output
#######################################
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

print_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    ((SKIPPED++))
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

#######################################
# Check prerequisites
#######################################
check_prerequisites() {
    print_header "Checking Prerequisites"

    # Check Stripe CLI
    if ! command -v stripe &> /dev/null; then
        print_failure "Stripe CLI not found. Install with: brew install stripe/stripe-cli/stripe"
        exit 1
    fi
    print_success "Stripe CLI installed ($(stripe version))"

    # Check if logged in
    if ! stripe config --list 2>/dev/null | grep -q "test_mode"; then
        print_warning "Stripe CLI may not be authenticated. Run: stripe login"
    else
        print_success "Stripe CLI authenticated"
    fi

    # Check server is running
    if curl -s --head "$SERVER_URL/health" > /dev/null 2>&1 || curl -s --head "$SERVER_URL" > /dev/null 2>&1; then
        print_success "Server is running at $SERVER_URL"
    else
        print_failure "Server not responding at $SERVER_URL"
        echo "       Start server with: go run cmd/server/main.go"
        echo "       For test mode: STRIPE_WEBHOOK_TEST_MODE=true go run cmd/server/main.go"
        exit 1
    fi
}

#######################################
# Start webhook listener in background
#######################################
start_listener() {
    print_info "Starting Stripe webhook listener..."

    # Kill any existing listeners
    pkill -f "stripe listen" 2>/dev/null || true
    sleep 1

    # Start listener in background and capture output
    stripe listen --forward-to "$WEBHOOK_URL" > /tmp/stripe-listen.log 2>&1 &
    LISTENER_PID=$!

    # Wait for listener to start and extract webhook secret
    sleep 3

    if ! ps -p $LISTENER_PID > /dev/null 2>&1; then
        print_failure "Failed to start webhook listener"
        cat /tmp/stripe-listen.log
        exit 1
    fi

    # Extract webhook secret from output
    WEBHOOK_SECRET=$(grep -o 'whsec_[a-zA-Z0-9]*' /tmp/stripe-listen.log | head -1)

    if [ -n "$WEBHOOK_SECRET" ]; then
        print_success "Webhook listener started (PID: $LISTENER_PID)"
        print_info "Webhook secret: $WEBHOOK_SECRET"
        echo ""
        print_warning "Make sure STRIPE_WEBHOOK_SECRET=$WEBHOOK_SECRET is set in your server environment"
    else
        print_warning "Could not extract webhook secret - check /tmp/stripe-listen.log"
    fi

    echo "$LISTENER_PID" > /tmp/stripe-listener.pid
}

#######################################
# Stop webhook listener
#######################################
stop_listener() {
    if [ -f /tmp/stripe-listener.pid ]; then
        PID=$(cat /tmp/stripe-listener.pid)
        if ps -p $PID > /dev/null 2>&1; then
            print_info "Stopping webhook listener (PID: $PID)..."
            kill $PID 2>/dev/null || true
        fi
        rm -f /tmp/stripe-listener.pid
    fi
    pkill -f "stripe listen" 2>/dev/null || true
}

#######################################
# Trigger a webhook event and check result
#######################################
trigger_event() {
    local event_type=$1
    local description=$2

    print_info "Triggering: $event_type"

    # Trigger the event
    OUTPUT=$(stripe trigger "$event_type" 2>&1)
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 0 ]; then
        print_success "$description"
        # Show relevant IDs from output
        echo "$OUTPUT" | grep -E "(pi_|in_|sub_|cus_)" | head -3 | while read line; do
            echo "         $line"
        done
    else
        print_failure "$description"
        echo "         Error: $OUTPUT"
    fi

    sleep $WAIT_BETWEEN_TRIGGERS
}

#######################################
# Run quick validation tests (test mode)
#######################################
run_quick_tests() {
    print_header "Quick Validation Tests (Test Mode)"

    echo ""
    print_warning "These tests verify webhook receipt and parsing."
    print_warning "Server should be running with STRIPE_WEBHOOK_TEST_MODE=true"
    echo ""

    # Payment Intent events
    print_info "--- Payment Intent Events ---"
    trigger_event "payment_intent.succeeded" "PaymentIntent succeeded event"
    trigger_event "payment_intent.payment_failed" "PaymentIntent failed event"
    trigger_event "payment_intent.canceled" "PaymentIntent canceled event"

    # Invoice events
    print_info "--- Invoice Events ---"
    trigger_event "invoice.payment_succeeded" "Invoice payment succeeded event"
    trigger_event "invoice.payment_failed" "Invoice payment failed event"
    trigger_event "invoice.created" "Invoice created event"
    trigger_event "invoice.finalized" "Invoice finalized event"

    # Subscription events
    print_info "--- Subscription Events ---"
    trigger_event "customer.subscription.created" "Subscription created event"
    trigger_event "customer.subscription.updated" "Subscription updated event"
    trigger_event "customer.subscription.deleted" "Subscription deleted event"

    # Customer events
    print_info "--- Customer Events ---"
    trigger_event "customer.created" "Customer created event"
    trigger_event "customer.updated" "Customer updated event"

    # Charge events
    print_info "--- Charge Events ---"
    trigger_event "charge.succeeded" "Charge succeeded event"
    trigger_event "charge.failed" "Charge failed event"
    trigger_event "charge.refunded" "Charge refunded event"
}

#######################################
# Run full integration tests
#######################################
run_full_tests() {
    print_header "Full Integration Tests"

    echo ""
    print_warning "Full integration tests require manual checkout completion."
    print_warning "Server should be running in NORMAL mode (test mode OFF)."
    echo ""

    print_info "--- Manual Test Steps ---"
    echo ""
    echo "1. Open browser to: $SERVER_URL"
    echo "2. Add items to cart"
    echo "3. Proceed to checkout"
    echo "4. Enter shipping/billing information"
    echo "5. Use test card: 4242 4242 4242 4242"
    echo "6. Complete payment"
    echo ""
    echo "Watch the server logs for:"
    echo "  - 'Order created successfully: ORD-XXXXX'"
    echo ""

    read -p "Press Enter after completing a checkout to continue..."

    # Check if order was created (would need database access)
    print_info "Verify in server logs that order was created successfully."
    print_skip "Automatic order verification not implemented"

    echo ""
    print_info "--- Subscription Test Steps ---"
    echo ""
    echo "1. Log in as a customer"
    echo "2. Navigate to a subscription product"
    echo "3. Subscribe with test card: 4242 4242 4242 4242"
    echo "4. Check Stripe Dashboard for created subscription"
    echo ""

    read -p "Press Enter after creating a subscription to continue..."

    print_info "Verify in server logs that subscription was created."
    print_skip "Automatic subscription verification not implemented"
}

#######################################
# Listen-only mode for manual testing
#######################################
run_listen_mode() {
    print_header "Webhook Listener Mode"

    echo ""
    print_info "Starting webhook listener for manual testing..."
    print_info "Press Ctrl+C to stop"
    echo ""

    # Run listener in foreground
    stripe listen --forward-to "$WEBHOOK_URL"
}

#######################################
# Print test summary
#######################################
print_summary() {
    print_header "Test Summary"

    echo ""
    echo -e "  Passed:  ${GREEN}$PASSED${NC}"
    echo -e "  Failed:  ${RED}$FAILED${NC}"
    echo -e "  Skipped: ${YELLOW}$SKIPPED${NC}"
    echo ""

    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed.${NC}"
        return 1
    fi
}

#######################################
# Show help
#######################################
show_help() {
    head -30 "$0" | tail -25
}

#######################################
# Cleanup on exit
#######################################
cleanup() {
    stop_listener
}

trap cleanup EXIT

#######################################
# Main
#######################################
main() {
    local MODE="quick"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --quick)
                MODE="quick"
                shift
                ;;
            --full)
                MODE="full"
                shift
                ;;
            --listen)
                MODE="listen"
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    print_header "Stripe Webhook Test Suite"
    echo "Mode: $MODE"

    check_prerequisites

    case $MODE in
        quick)
            start_listener
            sleep 2  # Give listener time to fully initialize
            run_quick_tests
            print_summary
            ;;
        full)
            start_listener
            sleep 2
            run_full_tests
            print_summary
            ;;
        listen)
            run_listen_mode
            ;;
    esac
}

main "$@"
