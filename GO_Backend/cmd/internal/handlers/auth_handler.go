package handlers

import (
	"LickLib/cmd/internal/config"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type AuthHandler struct {
	cfg config.KeycloakConfig
}

func NewAuthHandler(cfg config.KeycloakConfig) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// @Summary      Login
// @Tags         auth
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        username  formData  string  true  "Username"
// @Param        password  formData  string  true  "Passwort"
// @Success      200  {object}  map[string]interface{}
// @Failure      401
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	//fmt.Printf("DEBUG Config: URL=%s, Realm=%s, ID=%s, Secret=%s\n",
	//h.cfg.URL, h.cfg.Realm, h.cfg.ClientID, h.cfg.ClientSecret)

	resp, err := http.PostForm(h.cfg.TokenUrl(), url.Values{
		"grant_type":    {"password"},
		"client_id":     {h.cfg.ClientID},
		"client_secret": {h.cfg.ClientSecret},
		"username":      {username},
		"password":      {password},
	})

	if err != nil {
		fmt.Println("Netzwerkfehler zu Keycloak:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// DAS HIER IST ENTSCHEIDEND: Was sagt Keycloak wirklich?
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Keycloak Fehler (Status %d): %s\n", resp.StatusCode, string(body))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body) // Reiche den echten Fehler an Bruno weiter
		return
	}

	// Wenn wir hier landen, war der Status 200 OK
	// 1. Body von Keycloak lesen
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Fehler beim Lesen der Keycloak-Antwort", http.StatusInternalServerError)
		return
	}

	// 2. Den Header auf JSON setzen
	w.Header().Set("Content-Type", "application/json")

	// 3. Statuscode 200 setzen (optional, da Standard)
	w.WriteHeader(http.StatusOK)

	// 4. Das JSON von Keycloak direkt an den User (Bruno) zurückgeben
	w.Write(body)
}
