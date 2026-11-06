package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/internal/translations"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func translationReviewDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TRANSLATION_REVIEW_TEST_DSN")
	if dsn == "" {
		t.Skip("set TRANSLATION_REVIEW_TEST_DSN to a disposable local translation_review_test database")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/translation_review_test" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Fatal("requires local translation_review_test")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := database.DB()
	t.Cleanup(func() { _ = pool.Close() })
	for _, r := range translations.Resources {
		if err = database.AutoMigrate(r.Model()); err != nil {
			t.Fatal(err)
		}
	}
	if err = database.AutoMigrate(&models.AuditLog{}, &models.User{}, &models.EventroSyncState{}); err != nil {
		t.Fatal(err)
	}
	if err = database.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='anon') THEN CREATE ROLE anon; END IF; IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='authenticated') THEN CREATE ROLE authenticated; END IF; END $$").Error; err != nil {
		t.Fatal(err)
	}
	if err = database.Exec("DROP FUNCTION IF EXISTS public.translation_delete_resource() CASCADE; DROP TABLE IF EXISTS translation_suggestions,translation_jobs,translation_usage_daily CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob("../supabase/migrations/*_ai_translation_review.sql")
	if err != nil || len(paths) != 1 {
		t.Fatal("migration missing")
	}
	migration, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Exec(string(migration)).Error; err != nil {
		t.Fatal(err)
	}
	previous := db.DB
	db.DB = database
	t.Cleanup(func() { db.DB = previous })
	t.Setenv("REVALIDATION_URL", "")
	t.Setenv("TRANSLATION_AI_ENABLED", "true")
	t.Setenv("TRANSLATION_WORKER_JOB_NAME", "")
	t.Setenv("jwtsecret_laganda", "translation-test-signing-secret")
	return database
}

type translationGenerator func(context.Context, translations.Job, translations.Source, []string, int) (translations.Result, error)

func (f translationGenerator) Generate(ctx context.Context, j translations.Job, s translations.Source, fields []string, n int) (translations.Result, error) {
	return f(ctx, j, s, fields, n)
}
func goodTranslation(_ context.Context, _ translations.Job, source translations.Source, fields []string, _ int) (translations.Result, error) {
	out := translations.Result{Fields: map[string]string{}, Tokens: 123}
	for _, id := range fields {
		out.Fields[id] = "Svenska: " + source.Fields[id]
	}
	return out, nil
}
func blogFixture(t *testing.T, database *gorm.DB, name string) models.Blogpost {
	t.Helper()
	post := models.Blogpost{Title: name, Text: "Original body", Author: "Test author"}
	if err := database.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	return post
}
func reviewRequest(t *testing.T, record translations.Record, action string, ids ...string) translations.Review {
	t.Helper()
	input := translations.Review{Action: action, Fingerprint: record.Fingerprint}
	for _, id := range ids {
		for _, f := range record.Fields {
			if f.ID == id {
				selection := translations.ReviewField{FieldID: id, Text: "Svenska: " + f.Source, TargetBefore: f.Published}
				if f.Suggestion != nil && f.Suggestion.Status == "draft" {
					selection.SuggestionID = f.Suggestion.ID
					selection.Version = f.Suggestion.Version
					selection.Text = f.Suggestion.Draft
				}
				input.Fields = append(input.Fields, selection)
			}
		}
	}
	return input
}
func httpTranslationReview(t *testing.T, resource string, id uint64, input translations.Review) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(input)
	request := auth.WithUserID(httptest.NewRequest(http.MethodPost, "/translations/review", bytes.NewReader(body)), 1)
	request = mux.SetURLVars(request, map[string]string{"resource": resource, "id": fmt.Sprint(id)})
	token, err := utils.GenerateAccessToken(1, nil, []string{resource + ".view", resource + ".edit"})
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	auth.Middleware(http.HandlerFunc(ReviewTranslations)).ServeHTTP(response, request)
	return response
}
func TestTranslationReviewDatabaseFlow(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	post := blogFixture(t, database, "Original title")
	id := uint64(post.ID)
	jobs, err := service.Generate(ctx, resource, []uint64{id}, 1, false)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("%v %v", jobs, err)
	}
	again, err := service.Generate(ctx, resource, []uint64{id}, 1, false)
	if err != nil || len(again) != 1 || jobs[0].ID != again[0].ID {
		t.Fatal("duplicate queue request")
	}
	if err = service.Run(ctx, translationGenerator(goodTranslation)); err != nil {
		t.Fatal(err)
	}
	var untouched models.Blogpost
	if err = database.First(&untouched, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if untouched.TitleSv != nil || untouched.TextSv != nil {
		t.Fatal("generation published text")
	}
	record, err := service.Record(ctx, resource, id)
	if err != nil {
		t.Fatal(err)
	}
	if record.Fields[0].Suggestion == nil || record.Fields[1].Suggestion == nil {
		t.Fatal("missing generated fields")
	}
	input := reviewRequest(t, record, "draft", "Title", "Text")
	input.Fields[0].Text = "Redigerad titel"
	response := httpTranslationReview(t, resource.ID, id, input)
	if response.Code != 200 {
		t.Fatalf("draft: %d %s", response.Code, response.Body.String())
	}
	record, _ = service.Record(ctx, resource, id)
	stale := reviewRequest(t, record, "approve", "Title", "Text")
	input = reviewRequest(t, record, "draft", "Title")
	input.Fields[0].Text = "Ny redigerad titel"
	response = httpTranslationReview(t, resource.ID, id, input)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	response = httpTranslationReview(t, resource.ID, id, stale)
	if response.Code != 409 {
		t.Fatalf("stale draft accepted: %d", response.Code)
	}
	record, _ = service.Record(ctx, resource, id)
	input = reviewRequest(t, record, "approve", "Title", "Text")
	response = httpTranslationReview(t, resource.ID, id, input)
	if response.Code != 200 {
		t.Fatalf("approve: %d %s", response.Code, response.Body.String())
	}
	var updated models.Blogpost
	if err = database.First(&updated, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.TitleSv == nil || *updated.TitleSv != "Ny redigerad titel" || updated.TextSv == nil || updated.Title != post.Title {
		t.Fatal("wrong publication")
	}
	var count int64
	database.Model(&models.AuditLog{}).Where("resource_type = 'blogposts' AND resource_id = ?", fmt.Sprint(id)).Count(&count)
	if count != 1 {
		t.Fatalf("expected one audit update, got %d", count)
	}
	// Explicit regeneration may propose a replacement for approved AI text;
	// it must not change the published value before another review.
	regenerated, err := service.Generate(ctx, resource, []uint64{id}, 1, true)
	if err != nil || len(regenerated) != 1 {
		t.Fatal("cannot explicitly regenerate approved AI text", err)
	}
	var stillPublished models.Blogpost
	if err = database.First(&stillPublished, id).Error; err != nil {
		t.Fatal(err)
	}
	if stillPublished.TitleSv == nil || *stillPublished.TitleSv != *updated.TitleSv {
		t.Fatal("regeneration changed published text")
	}
	if err = service.Cancel(ctx, resource, id); err != nil {
		t.Fatal(err)
	}
	// Original edits mark the published AI text as needing an update; never remove it.
	database.Model(&post).Update("title", "Changed English")
	record, _ = service.Record(ctx, resource, id)
	if record.Fields[0].Status != "needs_update" {
		t.Fatalf("status=%s", record.Fields[0].Status)
	}
	// A concurrent manual Swedish edit blocks old proposals.
	jobs, err = service.Generate(ctx, resource, []uint64{id}, 1, false)
	if err != nil || len(jobs) != 1 {
		t.Fatal("outdated generation", err)
	}
	if err = service.Run(ctx, translationGenerator(goodTranslation)); err != nil {
		t.Fatal(err)
	}
	record, _ = service.Record(ctx, resource, id)
	input = reviewRequest(t, record, "approve", "Title")
	database.Model(&post).Update("title_sv", "Manual correction")
	response = httpTranslationReview(t, resource.ID, id, input)
	if response.Code != 409 {
		t.Fatalf("manual edit overwritten: %d", response.Code)
	}
	// All new tables have RLS and no direct grants to public client roles.
	for _, table := range []string{"translation_jobs", "translation_suggestions", "translation_usage_daily"} {
		var rls bool
		database.Raw("SELECT relrowsecurity FROM pg_class WHERE oid = ?::regclass", "public."+table).Scan(&rls)
		var grant bool
		database.Raw("SELECT has_table_privilege('anon', ?, 'SELECT') OR has_table_privilege('authenticated', ?, 'UPDATE')", "public."+table, "public."+table).Scan(&grant)
		if !rls || grant {
			t.Fatalf("unprotected table %s", table)
		}
	}
}
func TestTranslationWorkerBudgetAndConcurrency(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	for i := 0; i < 6; i++ {
		post := blogFixture(t, database, fmt.Sprintf("Original %d", i))
		if _, err := service.Generate(ctx, resource, []uint64{uint64(post.ID)}, 1, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Create(&translations.Usage{Day: "2026-11-02", Used: 999999}).Error; err != nil {
		t.Fatal(err)
	}
	var calls, active, peak atomic.Int32
	generator := translationGenerator(func(ctx context.Context, j translations.Job, s translations.Source, fields []string, n int) (translations.Result, error) {
		calls.Add(1)
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		active.Add(-1)
		return goodTranslation(ctx, j, s, fields, n)
	})
	if err := service.Run(ctx, generator); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("budget exceeded")
	}
	var waiting int64
	database.Model(&translations.Job{}).Where("status = 'budget_wait'").Count(&waiting)
	if waiting != 6 {
		t.Fatalf("waiting=%d", waiting)
	}
	now = now.Add(24 * time.Hour)
	var wg sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); failures <- service.Run(ctx, generator) }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 6 || peak.Load() != 2 {
		t.Fatalf("calls=%d concurrent=%d", calls.Load(), peak.Load())
	}
	var usage translations.Usage
	database.First(&usage, "day = ?", "2026-11-03")
	if usage.Used != 6*123 || usage.Reserved != 0 {
		t.Fatalf("usage=%#v", usage)
	}
}
func TestTranslationExclusionsAndAtomicFailure(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	ctx := context.Background()
	exhibitors, _ := translations.Find("exhibitors")
	company := models.Exhibitor{Name: "Company", About: utils.StringPtr("Company-owned description")}
	database.Create(&company)
	if _, err := service.Generate(ctx, exhibitors, []uint64{uint64(company.ID)}, 1, false); err == nil {
		t.Fatal("company description sent to AI")
	}
	record, _ := service.Record(ctx, exhibitors, uint64(company.ID))
	response := httpTranslationReview(t, "exhibitors", uint64(company.ID), reviewRequest(t, record, "approve", "About"))
	if response.Code != 200 {
		t.Fatalf("manual translation blocked: %s", response.Body.String())
	}
	resource, _ := translations.Find("blogposts")
	post := blogFixture(t, database, "Original title")
	database.Model(&post).Update("text", "Go to https://armada.nu/")
	_, err := service.Generate(ctx, resource, []uint64{uint64(post.ID)}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	err = service.Run(ctx, translationGenerator(func(context.Context, translations.Job, translations.Source, []string, int) (translations.Result, error) {
		return translations.Result{Fields: map[string]string{"Title": "Svensk titel", "Text": "Go to https://evil.example/"}, Tokens: 300}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	database.Model(&translations.Suggestion{}).Where("resource='blogposts' AND record_id = ?", post.ID).Count(&count)
	if count != 0 {
		t.Fatal("partial response persisted")
	}
	post2 := blogFixture(t, database, "Another original")
	record, _ = service.Record(ctx, resource, uint64(post2.ID))
	input := reviewRequest(t, record, "approve", "Title", "Text")
	err = service.Review(ctx, resource, uint64(post2.ID), input, 1, func(tx *gorm.DB, before, after any, updates map[string]any) error {
		if err := tx.Model(before).Updates(updates).Error; err != nil {
			return err
		}
		return errors.New("audit failed")
	})
	if err == nil {
		t.Fatal("expected rollback")
	}
	var untouched models.Blogpost
	database.First(&untouched, post2.ID)
	if untouched.TitleSv != nil {
		t.Fatal("partial publication after failed audit")
	}
	// Generation disabled does not disable manual review.
	service.Config.Enabled = false
	if _, err = service.Generate(ctx, resource, []uint64{uint64(post2.ID)}, 1, false); !errors.Is(err, translations.ErrDisabled) {
		t.Fatal("disabled generation accepted")
	}
	// Permissions are checked independently of the resource's normal public GET.
	body := strings.NewReader(`{"ids":[1]}`)
	request := mux.SetURLVars(httptest.NewRequest("POST", "/translations/blogposts/generate", body), map[string]string{"resource": "blogposts"})
	token, _ := utils.GenerateAccessToken(1, nil, []string{"blogposts.view"})
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	auth.Middleware(http.HandlerFunc(GenerateTranslations)).ServeHTTP(response, request)
	if response.Code != 403 {
		t.Fatalf("unauthorized generation: %d", response.Code)
	}
}

func TestTranslationPolicyVersionsAndRejection(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	post := blogFixture(t, database, "Versioned original")
	id := uint64(post.ID)
	jobs, err := service.Generate(ctx, resource, []uint64{id}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a job enqueued before a new prompt was deployed.
	savedPrompt := "Saved policy for queued work"
	if err = database.Model(&translations.Job{}).Where("id = ?", jobs[0].ID).Updates(map[string]any{"prompt": savedPrompt, "prompt_version": "old-prompt"}).Error; err != nil {
		t.Fatal(err)
	}
	generator := translationGenerator(func(ctx context.Context, j translations.Job, s translations.Source, fields []string, n int) (translations.Result, error) {
		if j.Prompt != savedPrompt || j.PromptVersion != "old-prompt" || j.GlossaryVersion != translations.GlossaryVersion {
			t.Error("queued policy not preserved")
		}
		return goodTranslation(ctx, j, s, fields, n)
	})
	if err = service.Run(ctx, generator); err != nil {
		t.Fatal(err)
	}
	record, _ := service.Record(ctx, resource, id)
	if !record.Fields[0].OlderPolicy {
		t.Fatal("missing old policy warning")
	}
	input := reviewRequest(t, record, "reject", "Title", "Text")
	if response := httpTranslationReview(t, resource.ID, id, input); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if next, err := service.Generate(ctx, resource, []uint64{id}, 1, false); err != nil || len(next) != 0 {
		t.Fatal("rejected suggestions regenerated implicitly", err)
	}
	if next, err := service.Generate(ctx, resource, []uint64{id}, 1, true); err != nil || len(next) != 1 {
		t.Fatal("explicit regeneration failed", err)
	}
	if err = service.Run(ctx, translationGenerator(goodTranslation)); err != nil {
		t.Fatal(err)
	}
	var old int64
	database.Model(&translations.Suggestion{}).Where("record_id = ? AND status = 'rejected'", id).Count(&old)
	if old != 2 {
		t.Fatal("previous proposal versions lost")
	}
	// Manual review remains available while AI is off.
	t.Setenv("TRANSLATION_AI_ENABLED", "false")
	record, _ = service.Record(ctx, resource, id)
	if response := httpTranslationReview(t, resource.ID, id, reviewRequest(t, record, "approve", "Title", "Text")); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	// Deletion triggers work independently of the disabled worker.
	if err = database.Delete(&post).Error; err != nil {
		t.Fatal(err)
	}
	database.Model(&translations.Suggestion{}).Where("resource = ? AND record_id = ?", resource.ID, id).Count(&old)
	if old != 0 {
		t.Fatal("deleted resource proposals retained")
	}
}

func TestTranslationStaleQueueAndCancellation(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	post := blogFixture(t, database, "Queued original")
	id := uint64(post.ID)
	if _, err := service.Generate(ctx, resource, []uint64{id}, 1, false); err != nil {
		t.Fatal(err)
	}
	database.Model(&post).Update("title", "Changed before worker")
	var calls atomic.Int32
	generator := translationGenerator(func(ctx context.Context, j translations.Job, s translations.Source, f []string, n int) (translations.Result, error) {
		calls.Add(1)
		return goodTranslation(ctx, j, s, f, n)
	})
	if err := service.Run(ctx, generator); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("stale queued work sent to AI")
	}
	if _, err := service.Generate(ctx, resource, []uint64{id}, 1, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel(ctx, resource, id); err != nil {
		t.Fatal(err)
	}
	if err := service.Run(ctx, generator); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("cancelled work executed")
	}
}

func TestTranslationRetryUsageAndLeaseRecovery(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	post := blogFixture(t, database, "Retry original")
	jobs, err := service.Generate(ctx, resource, []uint64{uint64(post.ID)}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	generator := translationGenerator(func(context.Context, translations.Job, translations.Source, []string, int) (translations.Result, error) {
		calls++
		return translations.Result{Tokens: 100}, &translations.ProviderError{Category: "provider_http_503", Retry: true}
	})
	for range 3 {
		if err = service.Run(ctx, generator); err != nil {
			t.Fatal(err)
		}
		now = now.Add(5 * time.Minute)
	}
	var job translations.Job
	database.First(&job, jobs[0].ID)
	var usage translations.Usage
	database.First(&usage, "day = ?", "2026-11-02")
	if calls != 3 || job.Status != "failed" || usage.Used != 300 || usage.Reserved != 0 {
		t.Fatalf("calls=%d job=%s usage=%+v", calls, job.Status, usage)
	}
	// Lost executions charge their complete outstanding reservation before retry.
	jobs, err = service.Generate(ctx, resource, []uint64{uint64(post.ID)}, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	expired := now.Add(-time.Minute)
	database.Model(&jobs[0]).Updates(map[string]any{"status": "running", "slot": 0, "lease_until": expired, "attempts": 3, "reserved_tokens": 500, "reservation_day": "2026-11-02"})
	database.Model(&usage).Update("reserved", 500)
	if err = service.Run(ctx, generator); err != nil {
		t.Fatal(err)
	}
	database.First(&usage, "day = ?", "2026-11-02")
	if calls != 3 || usage.Used != 800 || usage.Reserved != 0 {
		t.Fatalf("lease recovery accounting: %+v", usage)
	}
}

func TestTranslationManualPublicationIsPreserved(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	post := blogFixture(t, database, "Manual original")
	id := uint64(post.ID)
	record, _ := service.Record(ctx, resource, id)
	input := reviewRequest(t, record, "approve", "Title", "Text")
	if response := httpTranslationReview(t, resource.ID, id, input); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	database.Model(&post).Update("title", "Changed original")
	jobs, err := service.Generate(ctx, resource, []uint64{id}, 1, true)
	if err != nil || len(jobs) != 0 {
		t.Fatal("manual translations became AI eligible", err)
	}
}

func TestTranslationRegenerationKeepsPublishedContent(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	ctx := context.Background()
	post := blogFixture(t, database, "Regeneration original")
	id := uint64(post.ID)
	if _, err := service.Generate(ctx, resource, []uint64{id}, 1, false); err != nil {
		t.Fatal(err)
	}
	if err := service.Run(ctx, translationGenerator(goodTranslation)); err != nil {
		t.Fatal(err)
	}
	record, _ := service.Record(ctx, resource, id)
	if response := httpTranslationReview(t, resource.ID, id, reviewRequest(t, record, "approve", "Title", "Text")); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var original models.Blogpost
	if err := database.First(&original, id).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		jobs, err := service.Generate(ctx, resource, []uint64{id}, 1, true)
		if err != nil || len(jobs) != 1 {
			t.Fatal("explicit replacement generation", err)
		}
		if err = service.Run(ctx, translationGenerator(goodTranslation)); err != nil {
			t.Fatal(err)
		}
		var current models.Blogpost
		if err = database.First(&current, id).Error; err != nil {
			t.Fatal(err)
		}
		if !translations.Equal(original.TitleSv, current.TitleSv) || !translations.Equal(original.TextSv, current.TextSv) {
			t.Fatal("regeneration published without review")
		}
	}
	var retained int64
	database.Model(&translations.Suggestion{}).Where("resource = ? AND record_id = ? AND status = 'superseded'", resource.ID, id).Count(&retained)
	if retained != 2 {
		t.Fatal("prior replacement drafts not retained")
	}
	// New proposals and rejections must not hide outdated published AI text.
	database.Model(&post).Update("title", "Changed after replacement draft")
	record, _ = service.Record(ctx, resource, id)
	if !record.Fields[0].PublishedOutdated || record.Fields[0].Status != "stale" {
		t.Fatal("published provenance hidden by replacement draft")
	}
	if response := httpTranslationReview(t, resource.ID, id, reviewRequest(t, record, "approve", "Title", "Text")); response.Code != 409 {
		t.Fatal("stale proposal approved", response.Body.String())
	}
	rejected := reviewRequest(t, record, "reject", "Title", "Text")
	if response := httpTranslationReview(t, resource.ID, id, rejected); response.Code != 200 {
		t.Fatal("cannot reject stale proposal", response.Body.String())
	}
	record, _ = service.Record(ctx, resource, id)
	if !record.Fields[0].PublishedOutdated || record.Fields[0].Status != "rejected" {
		t.Fatal("rejection lost publication provenance")
	}
	page, err := service.List(ctx, resource, "", "needs_update", 0, 10)
	if err != nil || page.Total != 1 {
		t.Fatal("outdated publication missing from review queue", err)
	}
}

func TestTranslationDispatchQueueCheck(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	resource, _ := translations.Find("blogposts")
	past, future := now.Add(-time.Minute), now.Add(time.Minute)
	slot0, slot1 := 0, 1
	add := func(status string, due time.Time, lease *time.Time, slot *int) {
		post := blogFixture(t, database, "Dispatcher original")
		jobs, err := service.Generate(ctx, resource, []uint64{uint64(post.ID)}, 1, false)
		if err != nil || len(jobs) != 1 {
			t.Fatal("create dispatch fixture", err)
		}
		if err = database.Model(&jobs[0]).Updates(map[string]any{"status": status, "not_before": due, "lease_until": lease, "slot": slot}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, sample := range []struct {
		name    string
		prepare func()
		want    bool
	}{
		{"empty", func() {}, false},
		{"queued but not due", func() { add("queued", future, nil, nil) }, false},
		{"budget waiting tomorrow", func() { add("budget_wait", future, nil, nil) }, false},
		{"terminal jobs", func() {
			for _, status := range []string{"completed", "failed", "cancelled", "stale"} {
				add(status, past, nil, nil)
			}
		}, false},
		{"queued ready", func() { add("queued", past, nil, nil) }, true},
		{"budget ready", func() { add("budget_wait", past, nil, nil) }, true},
		{"active leases only", func() { add("running", past, &future, &slot0) }, false},
		{"both slots busy", func() {
			add("running", past, &future, &slot0)
			add("running", past, &future, &slot1)
			add("queued", past, nil, nil)
		}, false},
		{"one free slot", func() { add("running", past, &future, &slot0); add("queued", past, nil, nil) }, true},
		{"expired lease without queued jobs", func() { add("running", past, &past, &slot0) }, true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			if err := database.Exec("TRUNCATE translation_jobs CASCADE").Error; err != nil {
				t.Fatal(err)
			}
			sample.prepare()
			calls := 0
			response := httptest.NewRecorder()
			dispatchTranslationJobs(response, httptest.NewRequest(http.MethodPost, "/internal/translations/dispatch", nil), service, func(context.Context) error { calls++; return nil })
			wantCode := http.StatusNoContent
			if sample.want {
				wantCode = http.StatusAccepted
			}
			if response.Code != wantCode || calls != map[bool]int{false: 0, true: 1}[sample.want] {
				t.Fatalf("code=%d, starts=%d", response.Code, calls)
			}
		})
	}
	// A failed start must return an error and leave work ready for the next retry.
	database.Exec("TRUNCATE translation_jobs CASCADE")
	add("queued", past, nil, nil)
	response := httptest.NewRecorder()
	dispatchTranslationJobs(response, httptest.NewRequest(http.MethodPost, "/", nil), service, func(context.Context) error { return errors.New("start unavailable") })
	if response.Code != http.StatusServiceUnavailable {
		t.Fatal("failed start acknowledged")
	}
	var queued int64
	database.Model(&translations.Job{}).Where("status='queued' AND not_before <= ?", now).Count(&queued)
	if queued != 1 {
		t.Fatal("failed dispatch mutated queue")
	}
	// Disabled generation and DB failure never launch a worker.
	service.Config.Enabled = false
	started, err := service.Dispatch(ctx, func(context.Context) error { t.Fatal("disabled dispatcher started worker"); return nil })
	if started || err != nil {
		t.Fatal("disabled dispatch", err)
	}
	service.Config.Enabled = true
	// Use a cancelled query to test storage failure without touching remote data.
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	started, err = service.Dispatch(cancelled, func(context.Context) error { t.Fatal("failed query started worker"); return nil })
	if started || err == nil {
		t.Fatal("query failure did not fail closed")
	}
}

func TestTranslationResourceSelection(t *testing.T) {
	database := translationReviewDatabase(t)
	if err := database.Exec("TRUNCATE blogposts CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRANSLATION_MAX_RECORDS", "7")
	posts := make([]models.Blogpost, 53)
	for i := range posts {
		posts[i] = models.Blogpost{Title: fmt.Sprintf("Selection %d", i), Text: "Body", Published: i%2 == 0}
	}
	if err := database.Create(&posts).Error; err != nil {
		t.Fatal(err)
	}
	request := func(resource string, permissions []string) *httptest.ResponseRecorder {
		token, err := utils.GenerateAccessToken(1, nil, permissions)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/translations/selection?q=not-found&page=99", nil)
		req = mux.SetURLVars(req, map[string]string{"resource": resource})
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		auth.Middleware(http.HandlerFunc(GetTranslationSelection)).ServeHTTP(response, req)
		return response
	}
	response := request("blogposts", []string{"blogposts.view", "blogposts.edit"})
	var result TranslationSelectionResponse
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &result) != nil || len(result.IDs) != 53 || result.BatchSize != 7 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("selection: %d %s", response.Code, response.Body)
	}
	for i, id := range result.IDs {
		if id != uint64(posts[i].ID) {
			t.Fatal("selection omitted or reordered a record")
		}
	}
	for _, permissions := range [][]string{{"blogposts.view"}, {"blogposts.edit"}, nil} {
		if response = request("blogposts", permissions); response.Code != http.StatusForbidden {
			t.Fatalf("unauthorized selection: %d", response.Code)
		}
	}
	for _, resource := range translations.Resources {
		response = request(resource.ID, []string{"*"})
		expected := http.StatusOK
		if !resource.AI {
			expected = http.StatusBadRequest
		}
		if response.Code != expected {
			t.Fatalf("resource %s: %d %s", resource.ID, response.Code, response.Body)
		}
	}
	if response = request("unknown", []string{"*"}); response.Code != http.StatusNotFound {
		t.Fatal("unknown resource accepted")
	}
	var count int64
	if err := database.Model(&translations.Job{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("selection created jobs", err)
	}
}

func TestTranslationFormattingSurvivesGenerationAndApproval(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("blogposts")
	original := "## Welcome\n\n**Meet** our hosts.\n\n- Register\n- Attend"
	translated := "## Valkommen\n\n**Mot** vara vardar.\n\n- Registrera\n- Delta"
	for _, broken := range []bool{false, true} {
		post := blogFixture(t, database, "Formatting original")
		if err := database.Model(&post).Update("text", original).Error; err != nil {
			t.Fatal(err)
		}
		jobs, err := service.Generate(context.Background(), resource, []uint64{uint64(post.ID)}, 1, false)
		if err != nil || len(jobs) != 1 {
			t.Fatal("queue formatting fixture", err)
		}
		value := translated
		if broken {
			value = strings.ReplaceAll(value, "**", "")
		}
		err = service.Run(context.Background(), translationGenerator(func(context.Context, translations.Job, translations.Source, []string, int) (translations.Result, error) {
			return translations.Result{Fields: map[string]string{"Title": "Svensk rubrik", "Text": value}, Tokens: 123}, nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		var job translations.Job
		if err = database.First(&job, jobs[0].ID).Error; err != nil {
			t.Fatal(err)
		}
		if broken {
			var count int64
			database.Model(&translations.Suggestion{}).Where("job_id = ?", job.ID).Count(&count)
			if job.Status != "failed" || count != 0 {
				t.Fatal("damaged formatting saved as a proposal")
			}
			continue
		}
		record, err := service.Record(context.Background(), resource, uint64(post.ID))
		if err != nil {
			t.Fatal(err)
		}
		response := httpTranslationReview(t, "blogposts", uint64(post.ID), reviewRequest(t, record, "approve", "Title", "Text"))
		if response.Code != http.StatusOK {
			t.Fatalf("approve formatting: %s", response.Body)
		}
		if err = database.First(&post, post.ID).Error; err != nil || post.TextSv == nil || *post.TextSv != translated || post.Text != original {
			t.Fatal("published formatting changed", err)
		}
		// A manual edit that removes formatting is also rejected at approval time.
		record, err = service.Record(context.Background(), resource, uint64(post.ID))
		if err != nil {
			t.Fatal(err)
		}
		manual := translations.Review{Action: "approve", Fingerprint: record.Fingerprint, Fields: []translations.ReviewField{{FieldID: "Text", Text: strings.ReplaceAll(translated, "**", ""), TargetBefore: post.TextSv}}}
		if response = httpTranslationReview(t, "blogposts", uint64(post.ID), manual); response.Code != http.StatusBadRequest {
			t.Fatalf("damaged manual edit accepted: %d", response.Code)
		}
	}
}

func TestTranslationHTMLFormattingSurvivesGenerationAndApproval(t *testing.T) {
	database := translationReviewDatabase(t)
	service := translations.New(database)
	resource, _ := translations.Find("events")
	original, translated := "<p>Welcome <strong>hosts</strong></p>", "<p>Valkommen <strong>vardar</strong></p>"
	event := models.Event{EventroID: "translation-html-formatting", Name: "Formatting event", Description: &original, EventStart: time.Now(), EventEnd: time.Now().Add(time.Hour)}
	// Tests share only the disposable DB; clear a prior fixture with this key.
	if err := database.Where("eventro_id = ?", event.EventroID).Delete(&models.Event{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	jobs, err := service.Generate(context.Background(), resource, []uint64{uint64(event.ID)}, 1, false)
	if err != nil || len(jobs) != 1 {
		t.Fatal("queue HTML fixture", err)
	}
	err = service.Run(context.Background(), translationGenerator(func(context.Context, translations.Job, translations.Source, []string, int) (translations.Result, error) {
		return translations.Result{Fields: map[string]string{"Name": "Evenemang", "Description": translated}, Tokens: 123}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.Record(context.Background(), resource, uint64(event.ID))
	if err != nil {
		t.Fatal(err)
	}
	response := httpTranslationReview(t, "events", uint64(event.ID), reviewRequest(t, record, "approve", "Name", "Description"))
	if response.Code != http.StatusOK {
		t.Fatalf("approve HTML formatting: %s", response.Body)
	}
	if err = database.First(&event, event.ID).Error; err != nil || event.DescriptionSv == nil || *event.DescriptionSv != translated || event.Description == nil || *event.Description != original {
		t.Fatal("published HTML formatting changed", err)
	}
	record, err = service.Record(context.Background(), resource, uint64(event.ID))
	if err != nil {
		t.Fatal(err)
	}
	manual := translations.Review{Action: "approve", Fingerprint: record.Fingerprint, Fields: []translations.ReviewField{{FieldID: "Description", Text: "<p>Valkommen vardar</p>", TargetBefore: event.DescriptionSv}}}
	if response = httpTranslationReview(t, "events", uint64(event.ID), manual); response.Code != http.StatusBadRequest {
		t.Fatalf("damaged HTML approved: %d", response.Code)
	}
}
