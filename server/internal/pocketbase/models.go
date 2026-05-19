package pocketbase

// PBRecord represents the common fields injected by PocketBase in every collection record.
type PBRecord struct {
	ID      string `json:"id,omitempty"` // Internal PocketBase ID (15 chars)
	Created string `json:"created,omitempty"`
	Updated string `json:"updated,omitempty"`
}

// GuildRecord maps the "guilds" collection schema.
type GuildRecord struct {
	PBRecord
	DiscordID             string `json:"discord_id"`
	Name                  string `json:"name"`
	Status                string `json:"status"` // select: active, archived
	AnnouncementChannelID string `json:"announcement_channel_id"`
}

// RoleRecord maps the "roles" collection schema.
type RoleRecord struct {
	PBRecord
	DiscordID        string `json:"discord_id"`
	Name             string `json:"name"`
	GuildID          string `json:"guild_id"` // relation to guilds (max 1)
	Shift            string `json:"shift"`
	CheckInTime      string `json:"check_in_time"`
	CheckoutCooldown int    `json:"checkout_cooldown"`
	IsMonitored      bool   `json:"is_monitored"`
	IsActive         bool   `json:"is_active"`
	SquadChannelID   string `json:"squad_channel_id"` // NOVO
}

// StudentRecord maps the "students" collection schema.
// Note: Includes "user_id" which is added programmatically during migration to associate with system users.
type StudentRecord struct {
	PBRecord
	DiscordID      string   `json:"discord_id"`
	Username       string   `json:"username"`
	Nickname       string   `json:"nickname"`
	RoleID         string   `json:"role_id"`            // relation to roles (max 1)
	SecondaryRoles []string `json:"secondary_roles"`     // relation to roles (multiple)
	GuildID        string   `json:"guild_id"`           // relation to guilds (max 1)
	ChannelID      string   `json:"channel_id"`         // Discord channel ID
	Status         string   `json:"status"`             // select: active, inactive, dropped
	Shift          string   `json:"shift"`              // select: morning, afternoon, night
	UserID         string   `json:"user_id,omitempty"` // relation to _pb_users_auth_ (max 1)
}

// ManagerRecord maps the "managers" collection schema.
// Note: Includes "user_id" which is added programmatically during migration to associate with system users.
type ManagerRecord struct {
	PBRecord
	DiscordID string   `json:"discord_id"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`               // select: admin, mentor, pedagogy
	Guilds    []string `json:"guilds"`             // relation to guilds (multiple)
	UserID    string   `json:"user_id,omitempty"` // relation to _pb_users_auth_ (max 1)
}

// AttendanceRecord maps the "attendances" collection schema.
type AttendanceRecord struct {
	PBRecord
	StudentID          string `json:"student_id"` // relation to students (max 1)
	Date               string `json:"date"`       // date
	ClockIn            string `json:"clock_in"`   // date
	ClockOut           string `json:"clock_out"`  // date
	Status             string `json:"status"`     // select: pending_checkout, completed, absent, justified, late
	Source             string `json:"source"`     // select: discord_bot, manual_override
	Notes              string `json:"notes"`      // notes
	CheckoutPromptSent bool   `json:"checkout_prompt_sent"` // NOVO
	Expand             struct {                             // NOVO: Para leitura via query
		Student struct {
			Username  string `json:"username"` // NOVO
			Nickname  string `json:"nickname"` // NOVO
			DiscordID string `json:"discord_id"` // NOVO: Para relatórios
			ChannelID string `json:"channel_id"`
			Role      struct {
				Name             string `json:"name"` // NOVO: Para relatórios
				CheckoutCooldown int    `json:"checkout_cooldown"`
			} `json:"role_id"`
		} `json:"student_id"`
	} `json:"expand"`
}

// ActivityRecord maps the "activities" collection schema.
type ActivityRecord struct {
	PBRecord
	GuildID     string `json:"guild_id"`    // relation to guilds (max 1)
	Title       string `json:"title"`       // title
	Description string `json:"description"` // HTML/Editor description
	Type        string `json:"type"`        // select: announcement, task, feedback_request
	DueDate     string `json:"due_date"`    // date
	Status      string `json:"status"`      // select: draft, published, archived
}

// BroadcastRecord maps the "broadcasts" collection schema.
type BroadcastRecord struct {
	PBRecord
	Content       string   `json:"content"`
	TargetType    string   `json:"target_type"`  // text: "public" or "private" (validated in handler, not in DB)
	TargetRoles   []string `json:"target_roles"` // json array of PocketBase role IDs
	Status        string   `json:"status"`       // text: scheduled, processing, completed, failed
	ScheduleTime  string   `json:"schedule_time"`
	GuildID       string   `json:"guild_id"`
	MetricsSent   int      `json:"metrics_sent"`
	MetricsErrors int      `json:"metrics_errors"`
}

