package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func FetchExhibitorsEventro(w http.ResponseWriter, r *http.Request) {
	req, _ := http.NewRequest("GET", "https://app.eventro.se/api/v1/fairs/"+os.Getenv("EVENTRO_FAIR_ID")+"/exhibitors/", nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("EVENTRO_API"))
	req.Header.Set("organization", os.Getenv("EVENTRO_ORG"))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Status:", resp.Status)
	fmt.Println("Body:", string(body))
}
