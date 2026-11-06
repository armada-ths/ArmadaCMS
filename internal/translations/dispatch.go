package translations

import "context"

// Dispatch is a read-only eligibility check shared by direct starts and recovery.
// A due retry or expired lease is work, even when there is no new queued record.
// Actual claims remain authoritative in the worker; another worker can consume
// a ready item between this snapshot and Cloud Run startup.
func (s *Service) Dispatch(ctx context.Context, launch func(context.Context) error) (bool, error) {
	if !s.Config.Enabled {
		return false, nil
	}
	now := s.Now().UTC()
	var runnable bool
	err := s.DB.WithContext(ctx).Raw(`SELECT
 EXISTS (SELECT 1 FROM translation_jobs WHERE status = 'running' AND lease_until < ?)
 OR (
  (SELECT count(*) FROM translation_jobs WHERE status = 'running' AND (lease_until >= ? OR lease_until IS NULL)) < 2
  AND EXISTS (SELECT 1 FROM translation_jobs WHERE status IN ('queued','budget_wait') AND not_before <= ?)
 )`, now, now, now).Scan(&runnable).Error
	if err != nil || !runnable {
		return false, err
	}
	if err = launch(ctx); err != nil {
		return false, err
	}
	return true, nil
}
