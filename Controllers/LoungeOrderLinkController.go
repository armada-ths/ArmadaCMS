package controllers

import (
	"ArmadaCMS/main/db"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const loungeAccessHeader = "X-Lounge-Access"

type loungeOrderLink struct {
	ID       int
	Token    string `json:"-"`
	Revision int64
}

func (loungeOrderLink) TableName() string {
	return "lounge_order_link_settings"
}

func (link loungeOrderLink) accepts(provided string) bool {
	return link.Token != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(link.Token)) == 1
}

func readLoungeOrderLink(query *gorm.DB, lock bool) (loungeOrderLink, error) {
	var link loungeOrderLink
	if lock {
		query = query.Clauses(clause.Locking{Strength: "SHARE"})
	}
	err := query.First(&link, 1).Error
	return link, err
}

func (link loungeOrderLink) url() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("ORDER_APP_BASE_URL")), "/")
	if base == "" {
		switch os.Getenv("LOUNGE_ORDER_ENVIRONMENT") {
		case "production":
			base = "https://order.armada.nu"
		case "staging":
			base = "https://staging.order.armada.nu"
		default:
			base = "http://localhost:8002"
		}
	}
	return base + "/?access=" + url.QueryEscape(link.Token)
}

// GetLoungeOrderLink returns the shareable order link once created.
// @Summary Get the lounge order link
// @Tags lounge-orders
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /lounge-orders/link [get]
func GetLoungeOrderLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	link, err := readLoungeOrderLink(db.DB, false)
	if err != nil {
		http.Error(w, "Could not load order link", http.StatusServiceUnavailable)
		return
	}
	response := map[string]string{"url": ""}
	if link.Token != "" {
		response["url"] = link.url()
	}
	writeJSONResponse(w, http.StatusOK, response)
}

// RotateLoungeOrderLink replaces the order link and revokes previous links.
// @Summary Rotate the lounge order link
// @Tags lounge-orders
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /lounge-orders/link/rotate [post]
func RotateLoungeOrderLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		http.Error(w, "Could not generate order link", http.StatusInternalServerError)
		return
	}
	token := "lo_" + hex.EncodeToString(secret[:])
	var after loungeOrderLink
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var before loungeOrderLink
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&before, 1).Error; err != nil {
			return err
		}
		after = before
		after.Token = token
		after.Revision++
		return updateWithAuditInTransaction(tx, r, "lounge-orders", "link", before, &after, func(tx *gorm.DB) error {
			return tx.Model(&before).Updates(map[string]any{"token": after.Token, "revision": after.Revision}).Error
		}, nil)
	})
	if err != nil {
		http.Error(w, "Could not rotate order link", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]string{"url": after.url()})
}

// GetLoungeOrderQR returns a PNG for the current order link.
// @Summary Get the lounge order QR code
// @Tags lounge-orders
// @Produce png
// @Success 200 {file} file
// @Security BearerAuth
// @Router /lounge-orders/link/qr [get]
func GetLoungeOrderQR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	link, err := readLoungeOrderLink(db.DB, false)
	if err != nil {
		http.Error(w, "Could not load order link", http.StatusServiceUnavailable)
		return
	}
	if link.Token == "" {
		http.NotFound(w, r)
		return
	}
	writeLinkQR(w, link.url(), "armada-order-qr", "")
}
