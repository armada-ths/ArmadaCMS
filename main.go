package main

import (
	controllers "ArmadaCMS/main/Controllers"
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
		log.Fatal("Error loading .env file")
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
		// Enter your models here
	)

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

	api := mux.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/login", controllers.Login)

	api.HandleFunc("/customusers", controllers.GetUsers).Methods("GET")
	api.HandleFunc("/customusers/{id}", controllers.GetUserByID).Methods("GET")
	api.HandleFunc("/customusers", controllers.CreateUser).Methods("POST")
	api.HandleFunc("/customusers/{id}", controllers.UpdateUser).Methods("PUT")
	api.HandleFunc("/customusers/{id}", controllers.DeleteUser).Methods("DELETE")

	api.HandleFunc("/profiles", controllers.GetProfiles).Methods("GET")
	api.HandleFunc("/profiles/{id}", controllers.GetProfileByID).Methods("GET")
	api.HandleFunc("/profiles", controllers.CreateProfile).Methods("POST")
	api.HandleFunc("/profiles/{id}", controllers.UpdateProfile).Methods("PUT")
	api.HandleFunc("/profiles/{id}", controllers.DeleteProfile).Methods("DELETE")

	api.HandleFunc("/teams", controllers.GetTeams).Methods("GET")
	api.HandleFunc("/teams/{id}", controllers.GetTeamByID).Methods("GET")
	api.HandleFunc("/teams", controllers.CreateTeam).Methods("POST")
	api.HandleFunc("/teams/{id}", controllers.UpdateTeam).Methods("PUT")
	api.HandleFunc("/teams/{id}", controllers.DeleteTeam).Methods("DELETE")

	api.HandleFunc("/timeline", controllers.GetTimelineDates).Methods("GET")
	api.HandleFunc("/timeline/{id}", controllers.GetTimelineDateByID).Methods("GET")
	api.HandleFunc("/timeline", controllers.CreateTimelineDate).Methods("POST")
	api.HandleFunc("/timeline/{id}", controllers.UpdateTimelineDate).Methods("PUT")
	api.HandleFunc("/timeline/{id}", controllers.DeleteTimelineDate).Methods("DELETE")

	api.HandleFunc("/programs", controllers.GetPrograms).Methods("GET")
	api.HandleFunc("/programs/{id}", controllers.GetProgramByID).Methods("GET")
	api.HandleFunc("/programs", controllers.CreateProgram).Methods("POST")
	api.HandleFunc("/programs/{id}", controllers.UpdateProgram).Methods("PUT")
	api.HandleFunc("/programs/{id}", controllers.DeleteProgram).Methods("DELETE")

	// industries
	api.HandleFunc("/industries", controllers.GetIndustries).Methods("GET")
	api.HandleFunc("/industries/{id}", controllers.GetIndustryByID).Methods("GET")
	api.HandleFunc("/industries", controllers.CreateIndustry).Methods("POST")
	api.HandleFunc("/industries/{id}", controllers.UpdateIndustry).Methods("PUT")
	api.HandleFunc("/industries/{id}", controllers.DeleteIndustry).Methods("DELETE")

	// events
	api.HandleFunc("/events", controllers.GetEvents).Methods("GET")
	api.HandleFunc("/events/{id}", controllers.GetEventByID).Methods("GET")
	api.HandleFunc("/events", controllers.CreateEvent).Methods("POST")
	api.HandleFunc("/events/{id}", controllers.UpdateEvent).Methods("PUT")
	api.HandleFunc("/events/{id}", controllers.DeleteEvent).Methods("DELETE")

	// exhibitors
	api.HandleFunc("/exhibitors", controllers.GetExhibitors).Methods("GET")
	api.HandleFunc("/exhibitors/{id}", controllers.GetExhibitorByID).Methods("GET")
	api.HandleFunc("/exhibitors", controllers.CreateExhibitor).Methods("POST")
	api.HandleFunc("/exhibitors/{id}", controllers.UpdateExhibitor).Methods("PUT")
	api.HandleFunc("/exhibitors/{id}", controllers.DeleteExhibitor).Methods("DELETE")

	// employments
	api.HandleFunc("/employments", controllers.GetEmployments).Methods("GET")
	api.HandleFunc("/employments/{id}", controllers.GetEmploymentByID).Methods("GET")
	api.HandleFunc("/employments", controllers.CreateEmployment).Methods("POST")
	api.HandleFunc("/employments/{id}", controllers.UpdateEmployment).Methods("PUT")
	api.HandleFunc("/employments/{id}", controllers.DeleteEmployment).Methods("DELETE")

	api.HandleFunc("/organization", controllers.GetOrganizationEndpoint).Methods("GET")

	api.HandleFunc("/test", controllers.Test).Methods("GET")

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
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
