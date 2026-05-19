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
// If dest is nil, the response body is discarded (fire-and-forget update).
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

	// Only decode the response body if the caller wants to capture the updated record.
	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			return fmt.Errorf("failed to decode updated record: %w", err)
		}
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

// FindAttendanceByStudentAndDate searches for an attendance record for a specific student and date.
// The dateStr should be in "YYYY-MM-DD" format.
// If found, it populates the dest pointer and returns true, nil.
// If not found, it returns false, nil.
func (r *Repository) FindAttendanceByStudentAndDate(studentID string, dateStr string, dest *AttendanceRecord) (bool, error) {
	filter := fmt.Sprintf("student_id='%s' && date>='%s 00:00:00' && date<='%s 23:59:59'", studentID, dateStr, dateStr)
	endpoint := fmt.Sprintf("api/collections/attendances/records?filter=%s&limit=1", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to query pocketbase for attendance: %w", err)
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
		return false, fmt.Errorf("failed to unmarshal attendance record: %w", err)
	}

	return true, nil
}

// FindRolesByGuild searches for all roles associated with the target Guild ID (PocketBase Record ID) in PocketBase.
func (r *Repository) FindRolesByGuild(guildID string) ([]RoleRecord, error) {
	filter := fmt.Sprintf("guild_id='%s'", guildID)
	endpoint := fmt.Sprintf("api/collections/roles/records?filter=%s&limit=200", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for guild roles: %w", err)
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

	var roles []RoleRecord
	if err := json.Unmarshal(listResp.Items, &roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles array: %w", err)
	}

	return roles, nil
}

// FindRolesByCheckInTime searches for active, monitored roles configured with the exact check-in time HH:MM (ex: "08:00").
func (r *Repository) FindRolesByCheckInTime(checkInTime string) ([]RoleRecord, error) {
	filter := fmt.Sprintf("check_in_time='%s' && is_monitored=true && is_active=true", checkInTime)
	endpoint := fmt.Sprintf("api/collections/roles/records?filter=%s&limit=200", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for scheduled roles: %w", err)
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

	var roles []RoleRecord
	if err := json.Unmarshal(listResp.Items, &roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles array: %w", err)
	}

	return roles, nil
}

// FindActiveStudentsByRole searches for students linked to a specific guild and role with status = 'active'.
func (r *Repository) FindActiveStudentsByRole(guildID string, roleID string) ([]StudentRecord, error) {
	filter := fmt.Sprintf("guild_id='%s' && role_id='%s' && status='active'", guildID, roleID)
	endpoint := fmt.Sprintf("api/collections/students/records?filter=%s&limit=200", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for active students: %w", err)
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

// FindStudentsByGuild fetches all student records in PocketBase associated with a specific Guild ID.
func (r *Repository) FindStudentsByGuild(guildID string) ([]StudentRecord, error) {
	filter := fmt.Sprintf("guild_id='%s'", guildID)
	endpoint := fmt.Sprintf("api/collections/students/records?filter=%s&limit=500", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for guild students: %w", err)
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

// FindByID retrieves a single record from a collection by its PocketBase internal 15-character ID.
func (r *Repository) FindByID(collection string, pbID string, dest interface{}) (bool, error) {
	endpoint := fmt.Sprintf("api/collections/%s/records/%s", collection, pbID)

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve record by id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("pocketbase retrieve record failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return false, fmt.Errorf("failed to decode retrieved record: %w", err)
	}

	return true, nil
}

// FindPendingCheckouts searches for all attendance records that have a status of 'pending_checkout'
// and where the checkout prompt has not been sent yet (checkout_prompt_sent = false).
// It expands 'student_id.role_id' to load the student channel ID and the role's cooldown value.
func (r *Repository) FindPendingCheckouts() ([]AttendanceRecord, error) {
	filter := "status='pending_checkout' && checkout_prompt_sent=false"
	expand := "student_id.role_id"
	endpoint := fmt.Sprintf("api/collections/attendances/records?filter=%s&expand=%s&limit=500", url.QueryEscape(filter), url.QueryEscape(expand))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for pending checkouts: %w", err)
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

	var attendances []AttendanceRecord
	if err := json.Unmarshal(listResp.Items, &attendances); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attendances array: %w", err)
	}

	return attendances, nil
}

// GetAttendancesByDateAndRole queries the PocketBase collections for all attendances in a specific guild,
// for a specific role and date range. It expands the 'student_id' relation to resolve student usernames and nicknames.
func (r *Repository) GetAttendancesByDateAndRole(guildID, roleID, targetDate string) ([]AttendanceRecord, error) {
	filter := fmt.Sprintf("student_id.guild_id='%s' && student_id.role_id='%s' && date>='%s 00:00:00' && date<='%s 23:59:59'", guildID, roleID, targetDate, targetDate)
	expand := "student_id"
	endpoint := fmt.Sprintf("api/collections/attendances/records?filter=%s&expand=%s&limit=500", url.QueryEscape(filter), url.QueryEscape(expand))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for daily attendances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pocketbase daily attendances query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	var attendances []AttendanceRecord
	if err := json.Unmarshal(listResp.Items, &attendances); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attendances array: %w", err)
	}

	return attendances, nil
}

// DeleteRecord deletes a single record from a collection by its PocketBase internal 15-character ID.
func (r *Repository) DeleteRecord(collection string, pbID string) error {
	endpoint := fmt.Sprintf("api/collections/%s/records/%s", collection, pbID)
	resp, err := r.client.SendRequest("DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to issue DELETE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pocketbase delete record failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// FindBroadcastsByGuild searches PocketBase for all broadcasts in a specific guild, sorted by schedule_time descending.
func (r *Repository) FindBroadcastsByGuild(guildID string) ([]BroadcastRecord, error) {
	filter := fmt.Sprintf("guild_id='%s'", guildID)
	endpoint := fmt.Sprintf("api/collections/broadcasts/records?filter=%s&sort=-schedule_time&limit=200", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for broadcasts: %w", err)
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

	var broadcasts []BroadcastRecord
	if err := json.Unmarshal(listResp.Items, &broadcasts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal broadcasts array: %w", err)
	}

	return broadcasts, nil
}

// FindScheduledBroadcastsBefore searches PocketBase for all scheduled broadcasts with a schedule time <= cutoff.
func (r *Repository) FindScheduledBroadcastsBefore(cutoff string) ([]BroadcastRecord, error) {
	filter := fmt.Sprintf("status='scheduled' && schedule_time<='%s'", cutoff)
	endpoint := fmt.Sprintf("api/collections/broadcasts/records?filter=%s&limit=200", url.QueryEscape(filter))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for scheduled broadcasts: %w", err)
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

	var broadcasts []BroadcastRecord
	if err := json.Unmarshal(listResp.Items, &broadcasts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal broadcasts array: %w", err)
	}

	return broadcasts, nil
}


// GetAttendancesByDateRange queries the PocketBase collections for all attendances in a specific guild,
// optionally filtered by role, within a specified date range. It expands the 'student_id.role_id' relation
// to resolve student details and their assigned role/squad name for BI reports.
func (r *Repository) GetAttendancesByDateRange(guildID, roleID, startDate, endDate string) ([]AttendanceRecord, error) {
	// Base filter by guild and date range
	filter := fmt.Sprintf("student_id.guild_id='%s' && date>='%s 00:00:00' && date<='%s 23:59:59'", guildID, startDate, endDate)
	
	// Optional role filter
	if roleID != "" {
		filter += fmt.Sprintf(" && student_id.role_id='%s'", roleID)
	}

	expand := "student_id.role_id"
	// Use a high limit to ensure we capture all records for a typical month's report
	endpoint := fmt.Sprintf("api/collections/attendances/records?filter=%s&expand=%s&limit=5000", url.QueryEscape(filter), url.QueryEscape(expand))

	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pocketbase for date range attendances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pocketbase date range attendances query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	var attendances []AttendanceRecord
	if err := json.Unmarshal(listResp.Items, &attendances); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attendances array: %w", err)
	}

	return attendances, nil
}

// CountRecords queries the collection and returns the total number of records.
func (r *Repository) CountRecords(collection string) (int, error) {
	endpoint := fmt.Sprintf("api/collections/%s/records?limit=1", collection)
	resp, err := r.client.SendRequest("GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to issue count request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("pocketbase count failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return 0, fmt.Errorf("failed to decode list response: %w", err)
	}

	return listResp.TotalItems, nil
}
