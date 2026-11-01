package controllers

import (
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestPhotoTranslationPersistence(t *testing.T) {
	testTranslationPersistence(t, []any{&models.PhotoEvent{}})
}

func TestPhotoTranslationIntegration(t *testing.T) {
	database := eventroSyncTestDatabase(t)
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("s", 48))
	t.Setenv("PHOTO_APP_BASE_URL", "https://photos.example.test")
	now := time.Now().UTC()
	event := models.PhotoEvent{Name: "English event", Description: "English description", NameSv: utils.StringPtr(" Svenskt event "), DescriptionSv: utils.StringPtr(" Svensk beskrivning "), UploadsOpenAt: now.Add(-time.Hour), UploadsCloseAt: now.Add(time.Hour), GalleryCloseAt: now.Add(24 * time.Hour), MaxPhotosPerGuest: 3}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	CreatePhotoEvent(response, httptest.NewRequest(http.MethodPost, "/photoevents", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	id := strconv.FormatUint(event.ID, 10)
	request := func(method, path string, body []byte) *http.Request {
		return mux.SetURLVars(httptest.NewRequest(method, path, bytes.NewReader(body)), map[string]string{"id": id})
	}
	token, err := utils.PhotoEventToken(event.ID, event.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	assertAccess := func(t *testing.T, nameSv, descriptionSv any) {
		t.Helper()
		response := httptest.NewRecorder()
		PhotoEventAccess(response, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/photos/"+token, nil), map[string]string{"token": token}))
		var access map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &access); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || access["name"] != event.Name || access["description"] != event.Description || access["nameSv"] != nameSv || access["descriptionSv"] != descriptionSv || access["remaining"] != float64(3) || access["uploads_open"] != true || access["gallery_open"] != true {
			t.Fatalf("unexpected public access: %d %s", response.Code, response.Body.String())
		}
	}
	assertAccess(t, "Svenskt event", "Svensk beskrivning")
	for _, test := range []struct {
		name                      string
		fields                    map[string]any
		wantName, wantDescription any
	}{
		{"omitted preserves both", nil, "Svenskt event", "Svensk beskrivning"},
		{"both update together", map[string]any{"nameSv": " Ny titel ", "descriptionSv": " Ny beskrivning "}, "Ny titel", "Ny beskrivning"},
		{"null and blank clear", map[string]any{"nameSv": nil, "descriptionSv": "  "}, nil, nil},
		{"restore both", map[string]any{"nameSv": "Titel", "descriptionSv": "Beskrivning"}, "Titel", "Beskrivning"},
		{"empty and null clear", map[string]any{"nameSv": "", "descriptionSv": nil}, nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			delete(payload, "nameSv")
			delete(payload, "descriptionSv")
			for key, value := range test.fields {
				payload[key] = value
			}
			body, err = json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			UpdatePhotoEvent(response, request(http.MethodPut, "/photoevents/"+id, body))
			if response.Code != http.StatusOK {
				t.Fatalf("update: %d %s", response.Code, response.Body.String())
			}
			assertAccess(t, test.wantName, test.wantDescription)
			var current models.PhotoEvent
			if err := database.First(&current, event.ID).Error; err != nil {
				t.Fatal(err)
			}
			if current.TokenVersion != event.TokenVersion || current.MaxPhotosPerGuest != 3 {
				t.Fatal("translation edit changed access or quota")
			}
		})
	}
	t.Run("link and rotation carry locale", func(t *testing.T) {
		response := httptest.NewRecorder()
		PhotoEventLink(response, request(http.MethodGet, "/photoevents/"+id+"/link?locale=sv-SE", nil))
		var link map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &link); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || link["url"] != "https://photos.example.test/sv/e/"+token {
			t.Fatalf("unexpected link: %s", response.Body.String())
		}
		response = httptest.NewRecorder()
		RotatePhotoEventToken(response, request(http.MethodPost, "/photoevents/"+id+"/rotate?locale=sv", nil))
		if err := json.Unmarshal(response.Body.Bytes(), &link); err != nil {
			t.Fatal(err)
		}
		rotated := strings.TrimPrefix(link["url"], "https://photos.example.test/sv/e/")
		rotatedID, version, err := utils.ParsePhotoEventToken(rotated)
		if response.Code != http.StatusOK || err != nil || rotatedID != event.ID || version != event.TokenVersion+1 {
			t.Fatalf("unexpected rotated link: %s (%v)", response.Body.String(), err)
		}
		response = httptest.NewRecorder()
		PhotoEventAccess(response, mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/photos/"+token, nil), map[string]string{"token": token}))
		if response.Code != http.StatusNotFound || response.Header().Get("X-Armada-Error-Code") != "event_is_unavailable" {
			t.Fatalf("old token accepted or untranslated error: %d %s", response.Code, response.Body.String())
		}
	})
}
