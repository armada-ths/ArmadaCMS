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

	mux.HandleFunc("/login", controllers.Login)

	mux.HandleFunc("/customusers", controllers.GetUsers).Methods("GET")
	mux.HandleFunc("/customusers/{id}", controllers.GetUserByID).Methods("GET")
	mux.HandleFunc("/customusers", controllers.CreateUser).Methods("POST")
	mux.HandleFunc("/customusers/{id}", controllers.UpdateUser).Methods("PUT")
	mux.HandleFunc("/customusers/{id}", controllers.DeleteUser).Methods("DELETE")

	mux.HandleFunc("/profiles", controllers.GetProfiles).Methods("GET")
	mux.HandleFunc("/profiles/{id}", controllers.GetProfileByID).Methods("GET")
	mux.HandleFunc("/profiles", controllers.CreateProfile).Methods("POST")
	mux.HandleFunc("/profiles/{id}", controllers.UpdateProfile).Methods("PUT")
	mux.HandleFunc("/profiles/{id}", controllers.DeleteProfile).Methods("DELETE")

	mux.HandleFunc("/teams", controllers.GetTeams).Methods("GET")
	mux.HandleFunc("/teams/{id}", controllers.GetTeamByID).Methods("GET")
	mux.HandleFunc("/teams", controllers.CreateTeam).Methods("POST")
	mux.HandleFunc("/teams/{id}", controllers.UpdateTeam).Methods("PUT")
	mux.HandleFunc("/teams/{id}", controllers.DeleteTeam).Methods("DELETE")

	mux.HandleFunc("/test", controllers.Test).Methods("GET")

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
