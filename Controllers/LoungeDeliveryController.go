package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const loungeQueueCondition = "delivery_status = 'pending' OR delivery_retry_requested"

var loungeSlackClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// DispatchLoungeOrders processes durable pending notifications from Cloud Scheduler.
// @Summary Dispatch pending lounge Slack notifications
// @Tags lounge-orders
// @Security SchedulerOIDC
// @Success 204
// @Router /internal/lounge-orders/dispatch [post]
func DispatchLoungeOrders(w http.ResponseWriter, r *http.Request) {
	if err := dispatchLoungeOrders(r, loungeSlackClient); err != nil {
		loungeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RunLocalLoungeDelivery provides the scheduler's equivalent for local CMS development.
func RunLocalLoungeDelivery(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/internal/lounge-orders/dispatch", nil)
			if err == nil {
				err = dispatchLoungeOrders(request, loungeSlackClient)
			}
			if err != nil {
				log.Printf("local lounge dispatch: %v", err)
			}
		}
	}
}

func dispatchLoungeOrders(r *http.Request, client *http.Client) error {
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	var ids []uint
	if err := db.DB.WithContext(ctx).Model(&models.LoungeOrder{}).
		Where(loungeQueueCondition).Order("id").Limit(32).Pluck("id", &ids).Error; err != nil {
		return err
	}

	var group errgroup.Group
	group.SetLimit(8)
	for _, id := range ids {
		group.Go(func() error {
			return dispatchLoungeOrder(r, client, id)
		})
	}
	return group.Wait()
}

func dispatchLoungeOrder(r *http.Request, client *http.Client, id uint) error {
	var order models.LoungeOrder
	err := db.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Where(loungeQueueCondition).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Preload("Lines", func(tx *gorm.DB) *gorm.DB { return tx.Order("position") }).
			First(&order).Error; err != nil {
			return err
		}
		before := order
		var token [32]byte
		if _, err := rand.Read(token[:]); err != nil {
			return err
		}
		expires := time.Now().Add(time.Minute)
		order.ClaimToken = hex.EncodeToString(token[:])
		order.ClaimExpiresAt = &expires
		order.DeliveryStatus, order.DeliveryError, order.DeliveryRetryRequested = "sending", "", false
		return updateLoungeOrderWithAudit(tx, r, before, &order, map[string]any{
			"claim_token": order.ClaimToken, "claim_expires_at": expires,
			"delivery_status": "sending", "delivery_error": "", "delivery_retry_requested": false,
		})
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	status, reason := sendLoungeSlack(r.Context(), client, order)
	// Record the outcome even if the scheduler disconnected during the Slack request.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
	defer cancel()
	err = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current models.LoungeOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND claim_token = ?", order.ID, order.ClaimToken).
			First(&current).Error; err != nil {
			return err
		}
		before := current
		current.DeliveryStatus, current.DeliveryError = status, reason
		return updateLoungeOrderWithAudit(tx, r.WithContext(ctx), before, &current, map[string]any{
			"delivery_status": status, "delivery_error": reason, "claim_token": "", "claim_expires_at": nil,
		})
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func sendLoungeSlack(ctx context.Context, client *http.Client, order models.LoungeOrder) (string, string) {
	live := os.Getenv("SLACK_ORDER_HOOK_URL")
	webhook := live
	if order.IsTest {
		webhook = os.Getenv("SLACK_ORDER_TEST_HOOK_URL")
	}
	address, err := url.Parse(webhook)
	if err != nil || address.Scheme != "https" || address.Host != "hooks.slack.com" ||
		address.User != nil || !strings.HasPrefix(address.Path, "/services/") ||
		address.RawPath != "" || address.RawQuery != "" || address.Fragment != "" ||
		(order.IsTest && webhook == live) {
		return "failed", "not_configured"
	}
	payload, err := json.Marshal(map[string]any{"text": loungeSlackText(order), "mrkdwn": true})
	if err != nil {
		return "failed", "invalid_payload"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address.String(), bytes.NewReader(payload))
	if err != nil {
		return "failed", "not_configured"
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return "unknown", "slack_unreachable"
	}
	defer response.Body.Close()
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		return "sent", ""
	case response.StatusCode >= 500:
		return "unknown", "slack_server_error"
	default:
		return "failed", "slack_rejected"
	}
}

func loungeSlackText(order models.LoungeOrder) string {
	escape := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	var message strings.Builder
	if order.IsTest {
		message.WriteString("*[STAGING TEST]*\n")
	}
	fmt.Fprintf(&message, "*Order #%d —— %s:*", order.ID, escape.Replace(order.CompanyName))
	var categories []uint
	linesByCategory := make(map[uint][]models.LoungeOrderLine)
	for _, line := range order.Lines {
		if len(linesByCategory[line.CategoryID]) == 0 {
			categories = append(categories, line.CategoryID)
		}
		linesByCategory[line.CategoryID] = append(linesByCategory[line.CategoryID], line)
	}
	for _, category := range categories {
		lines := linesByCategory[category]
		fmt.Fprintf(&message, "\n\n*_%s:_*", escape.Replace(lines[0].CategoryName))
		for _, line := range lines {
			fmt.Fprintf(&message, "\n- %s x %d", escape.Replace(line.ItemName), line.Quantity)
		}
	}
	return message.String()
}

// RetryLoungeDelivery queues an explicit retry of a failed or ambiguous notification.
// @Summary Queue a lounge Slack notification retry
// @Tags lounge-orders
// @Param id path int true "Order ID"
// @Success 202 {object} map[string]bool
// @Security BearerAuth
// @Router /lounge-orders/{id}/retry [post]
func RetryLoungeDelivery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, ok := loungePathID(w, r)
	if !ok {
		return
	}
	var conflict bool
	err := db.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var order models.LoungeOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return err
		}
		status := loungeDeliveryStatus(order, time.Now())
		if status == "sent" || status == "sending" {
			conflict = true
			return nil
		}
		if status == "pending" || order.DeliveryRetryRequested {
			return nil
		}
		before := order
		order.DeliveryRetryRequested = true
		return updateLoungeOrderWithAudit(tx, r, before, &order, map[string]any{"delivery_retry_requested": true})
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}
	if err != nil {
		loungeError(w, err)
		return
	}
	if conflict {
		http.Error(w, "Notification already sent or currently sending", http.StatusConflict)
		return
	}
	writeJSONResponse(w, http.StatusAccepted, map[string]bool{"queued": true})
}
