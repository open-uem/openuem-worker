package models

import (
	"context"
	"time"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/enrollmenttoken"
)

// ValidateEnrollmentToken validates an enrollment token and returns the associated tenant/site IDs
// Returns (tenantID, siteID, error) where siteID may be 0 if no site is associated
func (m *Model) ValidateEnrollmentToken(token string) (int, int, error) {
	if token == "" {
		return 0, 0, nil
	}

	ctx := context.Background()

	et, err := m.Client.EnrollmentToken.Query().
		Where(enrollmenttoken.Token(token)).
		WithTenant().
		WithSite().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, 0, nil // Token not found, not an error - just return 0,0
		}
		return 0, 0, err
	}

	// Check if token is active
	if !et.Active {
		return 0, 0, nil
	}

	// Check if token is expired
	if et.ExpiresAt != nil && et.ExpiresAt.Before(time.Now()) {
		return 0, 0, nil
	}

	// Check if max uses reached (0 = unlimited)
	if et.MaxUses > 0 && et.CurrentUses >= et.MaxUses {
		return 0, 0, nil
	}

	// Get tenant ID
	tenantID := 0
	if et.Edges.Tenant != nil {
		tenantID = et.Edges.Tenant.ID
	}

	// Get site ID (may be nil)
	siteID := 0
	if et.Edges.Site != nil {
		siteID = et.Edges.Site.ID
	}

	return tenantID, siteID, nil
}

// UseEnrollmentToken increments the current_uses counter for an enrollment token
func (m *Model) UseEnrollmentToken(token string) error {
	if token == "" {
		return nil
	}

	ctx := context.Background()

	_, err := m.Client.EnrollmentToken.Update().
		Where(enrollmenttoken.Token(token)).
		AddCurrentUses(1).
		Save(ctx)

	return err
}

// GetEnrollmentTokenInfo returns tenant and site IDs for a valid token, or falls back to provided IDs
func (m *Model) GetEnrollmentTokenInfo(token string, fallbackTenantID, fallbackSiteID int) (int, int, bool, error) {
	if token == "" {
		return fallbackTenantID, fallbackSiteID, false, nil
	}

	tenantID, siteID, err := m.ValidateEnrollmentToken(token)
	if err != nil {
		return fallbackTenantID, fallbackSiteID, false, err
	}

	// If token validation returned 0,0, use fallback
	if tenantID == 0 {
		return fallbackTenantID, fallbackSiteID, false, nil
	}

	// If site is not set in token, use fallback site if tenant matches
	if siteID == 0 {
		return tenantID, fallbackSiteID, true, nil
	}

	return tenantID, siteID, true, nil
}
