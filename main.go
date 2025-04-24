package main

import (
	controllers "ArmadaCMS/main/Controllers"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"fmt"
	"log"
	"net/http"

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

	fs := http.FileServer(http.Dir("./frontend/dist"))
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

	api.HandleFunc("/dates", controllers.GetFairDates).Methods("GET")
	api.HandleFunc("/events", controllers.Test).Methods("GET")
	api.HandleFunc("/exhibitors", controllers.Test).Methods("GET")
    api.HandleFunc("/organization", controllers.GetOrganization).Methods("GET")

    api.HandleFunc("/test", controllers.Test).Methods("GET")

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
