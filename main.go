package main

import (
	controllers "ArmadaCMS/main/Controllers"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const port = 8080

func main() {
	fmt.Println("Hello, world.")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables directly")
	}

	db.ConnectDB()

	db.DB.AutoMigrate(
		models.User{},
		models.Blogpost{},
		models.Profile{},
		models.Team{},
		models.TimelineDate{},
		models.RefreshToken{},
		models.Industry{},
		models.Program{},
		models.Employment{},
		models.Exhibitor{},
		models.Event{},
		models.FairDateConfig{},
		models.FeatureFlag{},
		// Enter your models here
	)

	if err := controllers.SeedFeatureFlags(db.DB); err != nil {
		log.Printf("failed to seed feature flags: %v", err)
	}

	wrappedMux := CreateMuxClient()
	colonPort := fmt.Sprintf(":%d", port)
	fmt.Println("Server running on http://localhost" + colonPort)
	if err := http.ListenAndServe(colonPort, wrappedMux); err != nil {
		log.Fatal("Server error:", err)
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
func CreateControllers(mux *mux.Router) *mux.Router {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	buildDir := "./frontend/dist"
	fs := http.FileServer(http.Dir(buildDir))
	mux.PathPrefix("/admin/").Handler(http.StripPrefix("/admin/", fs))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	//refactor in future this is quity messy. // WD
	publicAPI := mux.PathPrefix("/api/v1").Subrouter()
	protectedAPI := mux.PathPrefix("/api/v1").Subrouter()
	protectedAPI.Use(auth.Middleware)
	publicAPI.HandleFunc("/login", controllers.Login)

	protectedAPI.HandleFunc("/customusers", controllers.GetUsers).Methods("GET")
	protectedAPI.HandleFunc("/customusers/{id}", controllers.GetUserByID).Methods("GET")
	protectedAPI.HandleFunc("/customusers", controllers.CreateUser).Methods("POST")
	protectedAPI.HandleFunc("/customusers/{id}", controllers.UpdateUser).Methods("PUT")
	protectedAPI.HandleFunc("/customusers/{id}", controllers.DeleteUser).Methods("DELETE")

	publicAPI.HandleFunc("/profiles", controllers.GetProfiles).Methods("GET")
	publicAPI.HandleFunc("/profiles/{id}", controllers.GetProfileByID).Methods("GET")
	protectedAPI.HandleFunc("/profiles", controllers.CreateProfile).Methods("POST")
	protectedAPI.HandleFunc("/profiles/{id}", controllers.UpdateProfile).Methods("PUT")
	protectedAPI.HandleFunc("/profiles/{id}", controllers.DeleteProfile).Methods("DELETE")

	publicAPI.HandleFunc("/teams", controllers.GetTeams).Methods("GET")
	publicAPI.HandleFunc("/teams/{id}", controllers.GetTeamByID).Methods("GET")
	protectedAPI.HandleFunc("/teams", controllers.CreateTeam).Methods("POST")
	protectedAPI.HandleFunc("/teams/{id}", controllers.UpdateTeam).Methods("PUT")
	protectedAPI.HandleFunc("/teams/{id}", controllers.DeleteTeam).Methods("DELETE")

	publicAPI.HandleFunc("/timeline", controllers.GetTimelineDates).Methods("GET")
	publicAPI.HandleFunc("/timeline/{id}", controllers.GetTimelineDateByID).Methods("GET")
	protectedAPI.HandleFunc("/timeline", controllers.CreateTimelineDate).Methods("POST")
	protectedAPI.HandleFunc("/timeline/{id}", controllers.UpdateTimelineDate).Methods("PUT")
	protectedAPI.HandleFunc("/timeline/{id}", controllers.DeleteTimelineDate).Methods("DELETE")

	publicAPI.HandleFunc("/programs", controllers.GetPrograms).Methods("GET")
	publicAPI.HandleFunc("/programs/{id}", controllers.GetProgramByID).Methods("GET")
	protectedAPI.HandleFunc("/programs", controllers.CreateProgram).Methods("POST")
	protectedAPI.HandleFunc("/programs/{id}", controllers.UpdateProgram).Methods("PUT")
	protectedAPI.HandleFunc("/programs/{id}", controllers.DeleteProgram).Methods("DELETE")

	// industries
	publicAPI.HandleFunc("/industries", controllers.GetIndustries).Methods("GET")
	publicAPI.HandleFunc("/industries/{id}", controllers.GetIndustryByID).Methods("GET")
	protectedAPI.HandleFunc("/industries", controllers.CreateIndustry).Methods("POST")
	protectedAPI.HandleFunc("/industries/{id}", controllers.UpdateIndustry).Methods("PUT")
	protectedAPI.HandleFunc("/industries/{id}", controllers.DeleteIndustry).Methods("DELETE")

	// events
	publicAPI.HandleFunc("/events", controllers.GetEvents).Methods("GET")
	publicAPI.HandleFunc("/events/{id}", controllers.GetEventByID).Methods("GET")
	protectedAPI.HandleFunc("/events", controllers.CreateEvent).Methods("POST")
	protectedAPI.HandleFunc("/events/{id}", controllers.UpdateEvent).Methods("PUT")
	protectedAPI.HandleFunc("/events/{id}", controllers.DeleteEvent).Methods("DELETE")

	// exhibitors
	publicAPI.HandleFunc("/exhibitors", controllers.GetExhibitors).Methods("GET")
	publicAPI.HandleFunc("/exhibitors/{id}", controllers.GetExhibitorByID).Methods("GET")
	protectedAPI.HandleFunc("/exhibitors", controllers.CreateExhibitor).Methods("POST")
	protectedAPI.HandleFunc("/exhibitors/{id}", controllers.UpdateExhibitor).Methods("PUT")
	protectedAPI.HandleFunc("/exhibitors/{id}", controllers.DeleteExhibitor).Methods("DELETE")

	// employments
	publicAPI.HandleFunc("/employments", controllers.GetEmployments).Methods("GET")
	publicAPI.HandleFunc("/employments/{id}", controllers.GetEmploymentByID).Methods("GET")
	protectedAPI.HandleFunc("/employments", controllers.CreateEmployment).Methods("POST")
	protectedAPI.HandleFunc("/employments/{id}", controllers.UpdateEmployment).Methods("PUT")
	protectedAPI.HandleFunc("/employments/{id}", controllers.DeleteEmployment).Methods("DELETE")

	// fairdates (admin CRUD)
	publicAPI.HandleFunc("/fairdates", controllers.GetFairDateConfigs).Methods("GET")
	publicAPI.HandleFunc("/fairdates/{id}", controllers.GetFairDateConfigByID).Methods("GET")
	protectedAPI.HandleFunc("/fairdates", controllers.CreateFairDateConfig).Methods("POST")
	protectedAPI.HandleFunc("/fairdates/{id}", controllers.UpdateFairDateConfig).Methods("PUT")
	protectedAPI.HandleFunc("/fairdates/{id}", controllers.DeleteFairDateConfig).Methods("DELETE")

	// feature flags
	publicAPI.HandleFunc("/featureflags", controllers.GetFeatureFlags).Methods("GET")
	publicAPI.HandleFunc("/featureflags/{id}", controllers.GetFeatureFlagByID).Methods("GET")
	protectedAPI.HandleFunc("/featureflags", controllers.CreateFeatureFlag).Methods("POST")
	protectedAPI.HandleFunc("/featureflags/{id}", controllers.UpdateFeatureFlag).Methods("PUT")
	protectedAPI.HandleFunc("/featureflags/{id}", controllers.DeleteFeatureFlag).Methods("DELETE")

	publicAPI.HandleFunc("/organization", controllers.GetOrganizationEndpoint).Methods("GET")
	publicAPI.HandleFunc("/dates", controllers.GetFairDates).Methods("GET")

	publicAPI.HandleFunc("/eventroexhibitors", controllers.FetchExhibitorsEventro).Methods("GET")
	publicAPI.HandleFunc("/eventroevents", controllers.FetchEventsEventro).Methods("GET")
	publicAPI.HandleFunc("/eventromembers", controllers.FetchMembersEventro).Methods("GET")

	publicAPI.HandleFunc("/test", controllers.Test).Methods("GET")

	mux.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(buildDir, r.URL.Path)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(buildDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Content-Range, Range")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
