// Package main is the entry point for ArmadaCMS.
//
// @title ArmadaCMS API
// @version 1.0
// @description REST API powering the THS Armada career fair website (armada.nu) and its admin panel.
// @contact.name THS Armada Development
// @contact.url https://github.com/armada-ths
//
// @BasePath /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer JWT token obtained from POST /api/v1/login. Format: "Bearer <token>"
package main

import (
	controllers "ArmadaCMS/main/Controllers"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	_ "ArmadaCMS/main/docs"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

var adminClientRouteSegments = map[string]struct{}{
	"customusers":        {},
	"profiles":           {},
	"teams":              {},
	"programs":           {},
	"industries":         {},
	"events":             {},
	"exhibitors":         {},
	"employments":        {},
	"fairdates":          {},
	"featureflags":       {},
	"roles":              {},
	"recruitmentperiods": {},
	"recruitmentroles":   {},
	"auditlogs":          {},
	"highlightcards":     {},
	"blogposts":          {},
	"login":              {},
}

func main() {
	fmt.Println("Hello, world.")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables directly")
	}

	if err := utils.ValidateJWTSecret(); err != nil {
		log.Fatal(err)
	}

	db.ConnectDB()

	if shouldRunAutoMigrate() {
		log.Println("DB_ENABLE_AUTOMIGRATE is enabled; running GORM AutoMigrate")
		if err := db.DB.AutoMigrate(
			models.AuditLog{},
			models.Role{},
			models.User{},
			models.Blogpost{},
			models.Profile{},
			models.Team{},
			models.RefreshToken{},
			models.Industry{},
			models.Program{},
			models.Employment{},
			models.Exhibitor{},
			models.Event{},
			models.RecruitmentPeriod{},
			models.RecruitmentRole{},
			models.FairDateConfig{},
			models.FeatureFlag{},
			models.HighlightCard{},
			// Enter your models here
		); err != nil {
			log.Fatalf("failed to run database migrations: %v", err)
		}
	} else {
		log.Println("DB_ENABLE_AUTOMIGRATE is disabled; expecting checked-in SQL migrations to own schema state")
	}

	if err := controllers.SeedRoles(db.DB); err != nil {
		log.Printf("failed to seed roles: %v", err)
	}

	if err := controllers.SeedAdminUser(db.DB); err != nil {
		log.Printf("failed to seed admin user: %v", err)
	}

	if err := controllers.SeedFeatureFlags(db.DB); err != nil {
		log.Printf("failed to seed feature flags: %v", err)
	}

	wrappedMux := CreateMuxClient()
	listenAddr := getListenAddr()
	fmt.Println("Server running on http://localhost" + listenAddr)
	if err := http.ListenAndServe(listenAddr, wrappedMux); err != nil {
		log.Fatal("Server error:", err)
	}
}

func getListenAddr() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	return fmt.Sprintf(":%s", port)
}

func shouldRunAutoMigrate() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("DB_ENABLE_AUTOMIGRATE")))
	if value == "" {
		return true
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Printf("invalid DB_ENABLE_AUTOMIGRATE=%q, defaulting to enabled", value)
		return true
	}
}

func CreateMuxClient() http.Handler {
	mux := mux.NewRouter()

	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("REQUEST: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	mux = CreateControllers(mux)
	wrappedMux := HandleCORS(mux)
	return wrappedMux
}
func isKnownAdminClientPath(relPath string) bool {
	trimmed := strings.Trim(relPath, "/")
	if trimmed == "" || trimmed == "." {
		return true
	}

	if filepath.Ext(trimmed) != "" {
		return true
	}

	firstSegment := strings.SplitN(trimmed, "/", 2)[0]
	_, ok := adminClientRouteSegments[firstSegment]
	return ok
}

func CreateControllers(mux *mux.Router) *mux.Router {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	buildDir := "./frontend/dist"

	serveAdminIndex := func(w http.ResponseWriter, r *http.Request, status int) {
		indexPath := filepath.Join(buildDir, "index.html")
		indexContent, err := os.ReadFile(indexPath)
		if err != nil {
			http.Error(w, "Failed to load admin app", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			_, _ = w.Write(indexContent)
		}
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		serveAdminIndex(w, r, http.StatusOK)
	})
	mux.PathPrefix("/admin/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/admin/")
		if relPath == "" || relPath == "." {
			serveAdminIndex(w, r, http.StatusOK)
			return
		}

		assetPath := filepath.Join(buildDir, filepath.FromSlash(relPath))
		if info, err := os.Stat(assetPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, assetPath)
			return
		}

		if !isKnownAdminClientPath(relPath) {
			serveAdminIndex(w, r, http.StatusNotFound)
			return
		}

		serveAdminIndex(w, r, http.StatusOK)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Swagger UI — served at /swagger/index.html
	mux.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	//refactor in future this is quity messy. // WD
	publicAPI := mux.PathPrefix("/api/v1").Subrouter()
	protectedAPI := mux.PathPrefix("/api/v1").Subrouter()
	protectedAPI.Use(auth.Middleware)
	publicAPI.HandleFunc("/login", controllers.Login)
	publicAPI.HandleFunc("/refreshAccessToken", controllers.RefreshAccessToken)

	// Current user info (for frontend permissions)
	protectedAPI.HandleFunc("/me", controllers.GetMe).Methods("GET")

	// Change own password — standalone permission
	protectedAPI.HandleFunc("/me/password", auth.RequirePermission("customusers.changeownpassword", controllers.ChangeOwnPassword)).Methods("PUT")

	// Users — admin only
	protectedAPI.HandleFunc("/customusers", auth.RequirePermission("customusers.view", controllers.GetUsers)).Methods("GET")
	protectedAPI.HandleFunc("/customusers/{id}", auth.RequirePermission("customusers.view", controllers.GetUserByID)).Methods("GET")
	protectedAPI.HandleFunc("/customusers", auth.RequirePermission("customusers.create", controllers.CreateUser)).Methods("POST")
	protectedAPI.HandleFunc("/customusers/{id}", auth.RequirePermission("customusers.edit", controllers.UpdateUser)).Methods("PUT")
	protectedAPI.HandleFunc("/customusers/{id}", auth.RequirePermission("customusers.delete", controllers.DeleteUser)).Methods("DELETE")

	// Roles — admin only
	protectedAPI.HandleFunc("/roles", auth.RequirePermission("roles.view", controllers.GetRoles)).Methods("GET")
	protectedAPI.HandleFunc("/roles/{id}", auth.RequirePermission("roles.view", controllers.GetRoleByID)).Methods("GET")
	protectedAPI.HandleFunc("/roles", auth.RequirePermission("roles.create", controllers.CreateRole)).Methods("POST")
	protectedAPI.HandleFunc("/roles/{id}", auth.RequirePermission("roles.edit", controllers.UpdateRole)).Methods("PUT")
	protectedAPI.HandleFunc("/roles/{id}", auth.RequirePermission("roles.delete", controllers.DeleteRole)).Methods("DELETE")

	publicAPI.HandleFunc("/profiles", controllers.GetProfiles).Methods("GET")
	publicAPI.HandleFunc("/profiles/{id}", controllers.GetProfileByID).Methods("GET")
	protectedAPI.HandleFunc("/profiles", auth.RequirePermission("profiles.create", controllers.CreateProfile)).Methods("POST")
	protectedAPI.HandleFunc("/profiles/{id}", auth.RequirePermission("profiles.edit", controllers.UpdateProfile)).Methods("PUT")
	protectedAPI.HandleFunc("/profiles/{id}", auth.RequirePermission("profiles.delete", controllers.DeleteProfile)).Methods("DELETE")

	publicAPI.HandleFunc("/teams", controllers.GetTeams).Methods("GET")
	publicAPI.HandleFunc("/teams/{id}", controllers.GetTeamByID).Methods("GET")
	protectedAPI.HandleFunc("/teams", auth.RequirePermission("teams.create", controllers.CreateTeam)).Methods("POST")
	protectedAPI.HandleFunc("/teams/{id}", auth.RequirePermission("teams.edit", controllers.UpdateTeam)).Methods("PUT")
	protectedAPI.HandleFunc("/teams/{id}", auth.RequirePermission("teams.delete", controllers.DeleteTeam)).Methods("DELETE")

	publicAPI.HandleFunc("/programs", controllers.GetPrograms).Methods("GET")
	publicAPI.HandleFunc("/programs/{id}", controllers.GetProgramByID).Methods("GET")
	protectedAPI.HandleFunc("/programs", auth.RequirePermission("programs.create", controllers.CreateProgram)).Methods("POST")
	protectedAPI.HandleFunc("/programs/{id}", auth.RequirePermission("programs.edit", controllers.UpdateProgram)).Methods("PUT")
	protectedAPI.HandleFunc("/programs/{id}", auth.RequirePermission("programs.delete", controllers.DeleteProgram)).Methods("DELETE")

	// industries
	publicAPI.HandleFunc("/industries", controllers.GetIndustries).Methods("GET")
	publicAPI.HandleFunc("/industries/{id}", controllers.GetIndustryByID).Methods("GET")
	protectedAPI.HandleFunc("/industries", auth.RequirePermission("industries.create", controllers.CreateIndustry)).Methods("POST")
	protectedAPI.HandleFunc("/industries/{id}", auth.RequirePermission("industries.edit", controllers.UpdateIndustry)).Methods("PUT")
	protectedAPI.HandleFunc("/industries/{id}", auth.RequirePermission("industries.delete", controllers.DeleteIndustry)).Methods("DELETE")

	// events
	publicAPI.HandleFunc("/events", controllers.GetEvents).Methods("GET")
	publicAPI.HandleFunc("/events/{id}", controllers.GetEventByID).Methods("GET")
	protectedAPI.HandleFunc("/events", auth.RequirePermission("events.create", controllers.CreateEvent)).Methods("POST")
	protectedAPI.HandleFunc("/events/{id}", auth.RequirePermission("events.edit", controllers.UpdateEvent)).Methods("PUT")
	protectedAPI.HandleFunc("/events/{id}", auth.RequirePermission("events.delete", controllers.DeleteEvent)).Methods("DELETE")

	// exhibitors
	publicAPI.HandleFunc("/exhibitors", controllers.GetExhibitors).Methods("GET")
	publicAPI.HandleFunc("/exhibitors/{id}", controllers.GetExhibitorByID).Methods("GET")
	protectedAPI.HandleFunc("/exhibitors", auth.RequirePermission("exhibitors.create", controllers.CreateExhibitor)).Methods("POST")
	protectedAPI.HandleFunc("/exhibitors/{id}", auth.RequirePermission("exhibitors.edit", controllers.UpdateExhibitor)).Methods("PUT")
	protectedAPI.HandleFunc("/exhibitors/{id}", auth.RequirePermission("exhibitors.delete", controllers.DeleteExhibitor)).Methods("DELETE")

	// employments
	publicAPI.HandleFunc("/employments", controllers.GetEmployments).Methods("GET")
	publicAPI.HandleFunc("/employments/{id}", controllers.GetEmploymentByID).Methods("GET")
	protectedAPI.HandleFunc("/employments", auth.RequirePermission("employments.create", controllers.CreateEmployment)).Methods("POST")
	protectedAPI.HandleFunc("/employments/{id}", auth.RequirePermission("employments.edit", controllers.UpdateEmployment)).Methods("PUT")
	protectedAPI.HandleFunc("/employments/{id}", auth.RequirePermission("employments.delete", controllers.DeleteEmployment)).Methods("DELETE")

	// fairdates (admin CRUD)
	publicAPI.HandleFunc("/fairdates", controllers.GetFairDateConfigs).Methods("GET")
	publicAPI.HandleFunc("/fairdates/{id}", controllers.GetFairDateConfigByID).Methods("GET")
	protectedAPI.HandleFunc("/fairdates", auth.RequirePermission("fairdates.create", controllers.CreateFairDateConfig)).Methods("POST")
	protectedAPI.HandleFunc("/fairdates/{id}", auth.RequirePermission("fairdates.edit", controllers.UpdateFairDateConfig)).Methods("PUT")
	protectedAPI.HandleFunc("/fairdates/{id}", auth.RequirePermission("fairdates.delete", controllers.DeleteFairDateConfig)).Methods("DELETE")

	// feature flags
	publicAPI.HandleFunc("/featureflags", controllers.GetFeatureFlags).Methods("GET")
	publicAPI.HandleFunc("/featureflags/{id}", controllers.GetFeatureFlagByID).Methods("GET")
	protectedAPI.HandleFunc("/featureflags", auth.RequirePermission("featureflags.create", controllers.CreateFeatureFlag)).Methods("POST")
	protectedAPI.HandleFunc("/featureflags/{id}", auth.RequirePermission("featureflags.edit", controllers.UpdateFeatureFlag)).Methods("PUT")
	protectedAPI.HandleFunc("/featureflags/{id}", auth.RequirePermission("featureflags.delete", controllers.DeleteFeatureFlag)).Methods("DELETE")

	// highlight cards
	publicAPI.HandleFunc("/highlightcards", controllers.GetHighlightCards).Methods("GET")
	publicAPI.HandleFunc("/highlightcards/{id}", controllers.GetHighlightCardByID).Methods("GET")
	protectedAPI.HandleFunc("/highlightcards", auth.RequirePermission("highlightcards.create", controllers.CreateHighlightCard)).Methods("POST")
	protectedAPI.HandleFunc("/highlightcards/{id}", auth.RequirePermission("highlightcards.edit", controllers.UpdateHighlightCard)).Methods("PUT")
	protectedAPI.HandleFunc("/highlightcards/{id}", auth.RequirePermission("highlightcards.delete", controllers.DeleteHighlightCard)).Methods("DELETE")

	// blogposts
	publicAPI.HandleFunc("/blogposts", controllers.GetBlogposts).Methods("GET")
	publicAPI.HandleFunc("/blogposts/{id}", controllers.GetBlogpostByID).Methods("GET")
	protectedAPI.HandleFunc("/blogposts", auth.RequirePermission("blogposts.create", controllers.CreateBlogpost)).Methods("POST")
	protectedAPI.HandleFunc("/blogposts/upload", auth.RequirePermission("blogposts.create", controllers.UploadBlogImage)).Methods("POST")
	protectedAPI.HandleFunc("/blogposts/{id}", auth.RequirePermission("blogposts.edit", controllers.UpdateBlogpost)).Methods("PUT")
	protectedAPI.HandleFunc("/blogposts/{id}", auth.RequirePermission("blogposts.delete", controllers.DeleteBlogpost)).Methods("DELETE")

	publicAPI.HandleFunc("/organization", controllers.GetOrganizationEndpoint).Methods("GET")
	publicAPI.HandleFunc("/dates", controllers.GetFairDates).Methods("GET")

	protectedAPI.HandleFunc("/eventroexhibitors", auth.RequirePermission("eventrosync.access", controllers.FetchExhibitorsEventro)).Methods("GET")
	protectedAPI.HandleFunc("/eventroevents", auth.RequirePermission("eventrosync.access", controllers.FetchEventsEventro)).Methods("GET")
	protectedAPI.HandleFunc("/eventrofairdates", auth.RequirePermission("eventrosync.access", controllers.FetchFairDatesEventro)).Methods("GET")
	protectedAPI.HandleFunc("/eventrofairs", auth.RequirePermission("eventrosync.access", controllers.GetEventroFairs)).Methods("GET")
	protectedAPI.HandleFunc("/eventromembers", auth.RequirePermission("eventrosync.access", controllers.FetchMembersEventro)).Methods("GET")
	protectedAPI.HandleFunc("/eventrorecruitments", auth.RequirePermission("eventrosync.access", controllers.FetchRecruitmentsEventro)).Methods("GET")
	publicAPI.HandleFunc("/recruitment", controllers.GetRecruitment).Methods("GET")

	protectedAPI.HandleFunc("/recruitmentperiods", auth.RequirePermission("recruitmentperiods.view", controllers.GetRecruitmentPeriods)).Methods("GET")
	protectedAPI.HandleFunc("/recruitmentperiods/{id}", auth.RequirePermission("recruitmentperiods.view", controllers.GetRecruitmentPeriodByID)).Methods("GET")
	protectedAPI.HandleFunc("/recruitmentperiods", auth.RequirePermission("recruitmentperiods.create", controllers.CreateRecruitmentPeriod)).Methods("POST")
	protectedAPI.HandleFunc("/recruitmentperiods/{id}", auth.RequirePermission("recruitmentperiods.edit", controllers.UpdateRecruitmentPeriod)).Methods("PUT")
	protectedAPI.HandleFunc("/recruitmentperiods/{id}", auth.RequirePermission("recruitmentperiods.delete", controllers.DeleteRecruitmentPeriod)).Methods("DELETE")

	protectedAPI.HandleFunc("/recruitmentroles", auth.RequirePermission("recruitmentroles.view", controllers.GetRecruitmentRoles)).Methods("GET")
	protectedAPI.HandleFunc("/recruitmentroles/{id}", auth.RequirePermission("recruitmentroles.view", controllers.GetRecruitmentRoleByID)).Methods("GET")
	protectedAPI.HandleFunc("/recruitmentroles", auth.RequirePermission("recruitmentroles.create", controllers.CreateRecruitmentRole)).Methods("POST")
	protectedAPI.HandleFunc("/recruitmentroles/{id}", auth.RequirePermission("recruitmentroles.edit", controllers.UpdateRecruitmentRole)).Methods("PUT")
	protectedAPI.HandleFunc("/recruitmentroles/{id}", auth.RequirePermission("recruitmentroles.delete", controllers.DeleteRecruitmentRole)).Methods("DELETE")

	// audit logs — read-only
	protectedAPI.HandleFunc("/auditlogs", auth.RequirePermission("auditlogs.view", controllers.GetAuditLogs)).Methods("GET")
	protectedAPI.HandleFunc("/auditlogs/{id}", auth.RequirePermission("auditlogs.view", controllers.GetAuditLogByID)).Methods("GET")

	publicAPI.HandleFunc("/test", controllers.Test).Methods("GET")

	mux.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	return mux
}
func HandleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-RefreshAuthorization, Content-Range, Range")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
