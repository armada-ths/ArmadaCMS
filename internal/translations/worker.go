package translations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"sync"
	"time"
)

const queueLock int64 = 84264172

func dayAt(now time.Time) (string, time.Time) {
	zone, _ := time.LoadLocation("Europe/Stockholm")
	local := now.In(zone)
	return local.Format("2006-01-02"), time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, zone)
}
func queueMutex(tx *gorm.DB) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", queueLock).Error
}
func (s *Service) recoverExpired(tx *gorm.DB) error {
	var expired []Job
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status = 'running' AND lease_until < ?", s.Now().UTC()).Find(&expired).Error; err != nil {
		return err
	}
	for _, job := range expired {
		if job.ReservedTokens > 0 {
			if err := tx.Model(&Usage{}).Where("day = ?", job.ReservationDay).Updates(map[string]any{"used": gorm.Expr("used + ?", job.ReservedTokens), "reserved": gorm.Expr("reserved - ?", job.ReservedTokens)}).Error; err != nil {
				return err
			}
		}
		status := "queued"
		if job.Attempts >= 3 {
			status = "failed"
		}
		if err := tx.Model(&job).Updates(map[string]any{"status": status, "error": "worker_interrupted", "slot": nil, "lease_until": nil, "reserved_tokens": 0}).Error; err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) claim(ctx context.Context) (*Job, error) {
	var selected Job
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := queueMutex(tx); err != nil {
			return err
		}
		if err := s.recoverExpired(tx); err != nil {
			return err
		}
		var running []Job
		if err := tx.Where("status = 'running'").Find(&running).Error; err != nil {
			return err
		}
		slot := 0
		for _, j := range running {
			if j.Slot != nil && *j.Slot == 0 {
				slot = 1
			}
		}
		if len(running) >= 2 {
			return nil
		}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status IN ? AND not_before <= ?", []string{"queued", "budget_wait"}, s.Now().UTC()).Order("id ASC").First(&selected).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		lease := s.Now().UTC().Add(10 * time.Minute)
		selected.Status = "running"
		selected.Slot = &slot
		selected.LeaseUntil = &lease
		return tx.Save(&selected).Error
	})
	if err != nil || selected.ID == 0 {
		return nil, err
	}
	return &selected, nil
}
func (s *Service) reserve(ctx context.Context, job *Job, amount int64) (bool, error) {
	ok := false
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := queueMutex(tx); err != nil {
			return err
		}
		var current Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, job.ID).Error; err != nil {
			return err
		}
		if current.Status != "running" || current.Attempts != job.Attempts {
			return ErrConflict
		}
		if amount > s.Config.DailyBudget {
			return tx.Model(&current).Updates(map[string]any{"status": "failed", "error": "record_exceeds_daily_budget", "slot": nil, "lease_until": nil, "finished_at": s.Now().UTC()}).Error
		}
		day, next := dayAt(s.Now())
		usage := Usage{Day: day}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&usage).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&usage, "day = ?", day).Error; err != nil {
			return err
		}
		if usage.Used+usage.Reserved+amount > s.Config.DailyBudget {
			return tx.Model(&current).Updates(map[string]any{"status": "budget_wait", "not_before": next.UTC(), "slot": nil, "lease_until": nil}).Error
		}
		if err := tx.Model(&usage).Update("reserved", gorm.Expr("reserved + ?", amount)).Error; err != nil {
			return err
		}
		job.Attempts++
		job.ReservedTokens = amount
		job.ReservationDay = day
		ok = true
		return tx.Model(&current).Updates(map[string]any{"attempts": job.Attempts, "reserved_tokens": amount, "reservation_day": day}).Error
	})
	return ok, err
}
func (s *Service) finish(ctx context.Context, job Job, result Result, callErr error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := queueMutex(tx); err != nil {
			return err
		}
		var current Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, job.ID).Error; err != nil {
			return err
		}
		if current.Status != "running" || current.Attempts != job.Attempts {
			return ErrConflict
		}
		if job.ReservedTokens > 0 {
			spent := result.Tokens
			if spent <= 0 {
				spent = job.ReservedTokens
			}
			if err := tx.Model(&Usage{}).Where("day = ?", job.ReservationDay).Updates(map[string]any{"reserved": gorm.Expr("reserved - ?", job.ReservedTokens), "used": gorm.Expr("used + ?", spent)}).Error; err != nil {
				return err
			}
		}
		status, category := "completed", ""
		if callErr != nil {
			status = "failed"
			category = "invalid_translation_output"
			var provider *ProviderError
			if errors.As(callErr, &provider) {
				category = provider.Category
			}
			if retryable(callErr) && job.Attempts < 3 {
				status = "queued"
			}
		}
		if callErr == nil {
			r, err := Find(job.Resource)
			if err != nil {
				return err
			}
			m, err := r.Load(tx, job.RecordID, true)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = "stale"
				category = "record_deleted"
			} else if err != nil {
				return err
			} else {
				source := r.Snapshot(m, job.RecordID)
				var snapshot Source
				_ = json.Unmarshal([]byte(job.SourceJSON), &snapshot)
				var fields []string
				_ = json.Unmarshal([]byte(job.FieldIDs), &fields)
				if source.Fingerprint() != job.Fingerprint {
					status = "stale"
					category = "source_changed"
				}
				latest, err := latestSuggestions(tx, r.ID, job.RecordID)
				if err != nil {
					return err
				}
				for _, id := range fields {
					if !Equal(source.Targets[id], snapshot.Targets[id]) {
						status = "stale"
						category = "target_changed"
					}
					if previous, ok := latest[fmt.Sprintf("%d/%s", job.RecordID, id)]; ok && previous.UpdatedAt.After(job.CreatedAt) {
						status = "stale"
						category = "review_changed"
					}
				}
				validated := map[string][]string{}
				if status == "completed" && len(result.Fields) != len(fields) {
					status, category = "failed", "invalid_translation_output"
				}
				if status == "completed" {
					for _, id := range fields {
						f, fieldErr := r.Field(id)
						if fieldErr != nil {
							return fieldErr
						}
						warnings, validationErr := ValidateText(snapshot, f, result.Fields[id], job.Glossary)
						if validationErr != nil {
							status, category = "failed", "invalid_translation_output"
							break
						}
						validated[id] = warnings
					}
				}
				if status == "completed" {
					for _, id := range fields {
						warnings := validated[id]
						if err = tx.Model(&Suggestion{}).Where("resource = ? AND record_id = ? AND field_id = ? AND status = 'draft'", r.ID, job.RecordID, id).Updates(map[string]any{"status": "superseded", "version": gorm.Expr("version + 1")}).Error; err != nil {
							return err
						}
						suggestion := Suggestion{Resource: r.ID, RecordID: job.RecordID, FieldID: id, JobID: &job.ID, Version: 1, Model: job.Model, PromptVersion: job.PromptVersion, GlossaryVersion: job.GlossaryVersion, SourceFingerprint: job.Fingerprint, TargetBefore: snapshot.Targets[id], Generated: result.Fields[id], Draft: result.Fields[id], Status: "draft", Warnings: encode(warnings)}
						if err = tx.Create(&suggestion).Error; err != nil {
							return err
						}
					}
				}
			}
		}
		updates := map[string]any{"status": status, "error": category, "slot": nil, "lease_until": nil, "reserved_tokens": 0}
		if status == "queued" {
			updates["not_before"] = s.Now().UTC().Add(time.Duration(job.Attempts) * time.Minute)
		} else {
			updates["finished_at"] = s.Now().UTC()
		}
		return tx.Model(&current).Updates(updates).Error
	})
}
func (s *Service) process(ctx context.Context, job Job, generator Generator) error {
	var source Source
	var fields []string
	if json.Unmarshal([]byte(job.SourceJSON), &source) != nil || json.Unmarshal([]byte(job.FieldIDs), &fields) != nil {
		return s.finish(ctx, job, Result{}, errors.New("invalid_job"))
	}
	resource, resourceErr := Find(job.Resource)
	if resourceErr != nil || !resource.AI || source.Resource != job.Resource || len(fields) == 0 {
		return s.finish(ctx, job, Result{}, errors.New("AI resource not allowed"))
	}
	for _, id := range fields {
		if _, err := resource.Field(id); err != nil {
			return s.finish(ctx, job, Result{}, errors.New("invalid field"))
		}
	}
	current, loadErr := resource.Load(s.DB.WithContext(ctx), job.RecordID, false)
	stale := errors.Is(loadErr, gorm.ErrRecordNotFound)
	if loadErr != nil && !stale {
		return loadErr
	}
	if !stale {
		live := resource.Snapshot(current, job.RecordID)
		stale = live.Fingerprint() != job.Fingerprint
		for _, id := range fields {
			stale = stale || !Equal(live.Targets[id], source.Targets[id])
		}
	}
	if stale {
		return s.DB.WithContext(ctx).Model(&Job{}).Where("id = ? AND status = 'running' AND attempts = ?", job.ID, job.Attempts).Updates(map[string]any{"status": "stale", "error": "source_or_target_changed", "slot": nil, "lease_until": nil, "finished_at": s.Now().UTC()}).Error
	}
	ok, err := s.reserve(ctx, &job, reservation(job, source, fields, s.Config.MaxOutput))
	if err != nil || !ok {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, s.Config.RequestTimeout)
	result, callErr := generator.Generate(callCtx, job, source, fields, s.Config.MaxOutput)
	cancel()
	if callErr == nil {
		if len(result.Fields) != len(fields) {
			callErr = errors.New("wrong response fields")
		}
		for _, id := range fields {
			r, _ := Find(job.Resource)
			f, _ := r.Field(id)
			if _, exists := result.Fields[id]; !exists {
				callErr = errors.New("missing response field")
				break
			}
			if _, err := ValidateText(source, f, result.Fields[id], job.Glossary); err != nil {
				callErr = err
				break
			}
		}
	}
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer finishCancel()
	err = s.finish(finishCtx, job, result, callErr)
	category := "none"
	if callErr != nil {
		category = "invalid_translation_output"
		var provider *ProviderError
		if errors.As(callErr, &provider) {
			category = provider.Category
		}
	}
	log.Printf("translation job=%d model=%s prompt=%s glossary=%s tokens=%d category=%s", job.ID, job.Model, job.PromptVersion, job.GlossaryVersion, result.Tokens, category)
	return err
}
func (s *Service) cleanup(ctx context.Context) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, r := range Resources {
			statement := &gorm.Statement{DB: tx}
			if err := statement.Parse(r.Model()); err != nil {
				return err
			}
			missing := "resource = ? AND NOT EXISTS (SELECT 1 FROM " + statement.Schema.Table + " WHERE id = record_id)"
			if err := tx.Where(missing, r.ID).Delete(&Suggestion{}).Error; err != nil {
				return err
			}
			if err := tx.Where(missing+" AND status <> 'running'", r.ID).Delete(&Job{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Service) Run(ctx context.Context, generator Generator) error {
	if !s.Config.Enabled {
		return nil
	}
	if err := s.cleanup(ctx); err != nil {
		return err
	}
	var wg sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if err := ctx.Err(); err != nil {
					failures <- err
					return
				}
				job, err := s.claim(ctx)
				if err != nil {
					failures <- err
					return
				}
				if job == nil {
					return
				}
				if err = s.process(ctx, *job, generator); err != nil {
					failures <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		return err
	}
	return s.cleanup(ctx)
}
