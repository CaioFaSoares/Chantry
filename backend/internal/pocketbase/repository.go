package pocketbase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Repository manages database CRUD operations via the PocketBase REST Client.
type Repository struct {
	client *Client
}

// NewRepository instantiates a new PocketBase Repository.
func NewRepository(client *Client) *Repository {
	return &Repository{
		client: client,
	}
}

// ListResponse defines the PocketBase standard envelope for multi-record list results.
type ListResponse struct {
	Page       int             `json:"page"`
	PerPage    int             `json:"perPage"`
	TotalItems int             `json:"totalItems"`
	TotalPages int             `json:"totalPages"`
	Items      json.RawMessage `json:"items"`
}

// FindFirstByDiscordID searches for a record inside the target collection using its Discord Snowflake ID.
// If found, it populates the dest interface (pointer to struct) and returns true, nil.
// If not found, it returns false, nil.
func (r *Repository) FindFirstByDiscordID(collection string, discordID string, dest interface{}) (bool, error) {
	filter := fmt.Sprintf("discord_id='%s'", discordID)
	endpoint := fmt.Sprintf("api/collections/%s/records?filter=%s&limit=1", collection, url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to query pocketbase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("pocketbase query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return false, fmt.Errorf("failed to decode list response: %w", err)
	}

	if listResp.TotalItems == 0 {
		return false, nil
	}

	// Parse the first item array element into target struct
	var items []json.RawMessage
	if err := json.Unmarshal(listResp.Items, &items); err != nil {
		return false, fmt.Errorf("failed to parse items array: %w", err)
	}

	if len(items) == 0 {
		return false, nil
	}

	if err := json.Unmarshal(items[0], dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal target record: %w", err)
	}

	return true, nil
}

// FindFirstByDiscordAndGuild searches for a record inside the target collection using its Discord Snowflake ID and the PocketBase Guild ID relation.
// If found, it populates the dest interface (pointer to struct) and returns true, nil.
// If not found, it returns false, nil.
func (r *Repository) FindFirstByDiscordAndGuild(collection string, discordID string, guildID string, dest interface{}) (bool, error) {
	filter := fmt.Sprintf("discord_id='%s' && guild_id='%s'", discordID, guildID)
	endpoint := fmt.Sprintf("api/collections/%s/records?filter=%s&limit=1", collection, url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to query pocketbase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("pocketbase query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return false, fmt.Errorf("failed to decode list response: %w", err)
	}

	if listResp.TotalItems == 0 {
		return false, nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(listResp.Items, &items); err != nil {
		return false, fmt.Errorf("failed to parse items array: %w", err)
	}

	if len(items) == 0 {
		return false, nil
	}

	if err := json.Unmarshal(items[0], dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal target record: %w", err)
	}

	return true, nil
}

// CreateRecord creates a new record inside the target collection and unmarshals the response with the generated PocketBase ID.
func (r *Repository) CreateRecord(collection string, data interface{}, dest interface{}) error {
	endpoint := fmt.Sprintf("api/collections/%s/records", collection)
	resp, err := r.client.SendRequest("POST", endpoint, data)
	if err != nil {
		return fmt.Errorf("failed to issue POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pocketbase create record failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("failed to decode created record: %w", err)
	}

	return nil
}

// UpdateRecord performs a partial update (PATCH) of an existing record using its unique 15-character PocketBase internal ID.
func (r *Repository) UpdateRecord(collection string, pbID string, data interface{}, dest interface{}) error {
	endpoint := fmt.Sprintf("api/collections/%s/records/%s", collection, pbID)
	resp, err := r.client.SendRequest("PATCH", endpoint, data)
	if err != nil {
		return fmt.Errorf("failed to issue PATCH request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pocketbase update record failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("failed to decode updated record: %w", err)
	}

	return nil
}

// FindManagersByGuild searches PocketBase for all manager records associated with a specific guild ID.
// Because the 'guilds' field in the 'managers' collection is a multiple-relation field,
// it performs a query using the '~' (contains) filter operator.
func (r *Repository) FindManagersByGuild(guildID string) ([]ManagerRecord, error) {
	filter := fmt.Sprintf("guilds~'%s'", guildID)
	endpoint := fmt.Sprintf("api/collections/managers/records?filter=%s", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for managers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pocketbase query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	var managers []ManagerRecord
	if err := json.Unmarshal(listResp.Items, &managers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managers array: %w", err)
	}

	return managers, nil
}

// FindStudentsPendingProvision searches for students in a specific guild and role who do not have a Discord channel provisioned yet.
// It uses limit=200 to ensure all pending students of a typical class are retrieved in a single batch.
func (r *Repository) FindStudentsPendingProvision(guildID string, roleID string) ([]StudentRecord, error) {
	filter := fmt.Sprintf("guild_id='%s' && role_id='%s' && channel_id=''", guildID, roleID)
	endpoint := fmt.Sprintf("api/collections/students/records?filter=%s&limit=200", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for pending students: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pocketbase query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	var students []StudentRecord
	if err := json.Unmarshal(listResp.Items, &students); err != nil {
		return nil, fmt.Errorf("failed to unmarshal students array: %w", err)
	}

	return students, nil
}
