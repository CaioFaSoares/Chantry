# Codebase Modelling Report: Data Schema

## Database: PocketBase (SQLite)
The application uses PocketBase for persistence. The schema is defined in `pb_schema.json` and mirrored in Go structs within `server/internal/pocketbase/models.go`.

## Key Collections

### `guilds`
Stores Discord server information.
- `discord_id`: Snowflake ID from Discord.
- `announcement_channel_id`: Target for global broadcasts.

### `roles`
Configuration for monitored roles (squads/classes).
- `shift`: morning, afternoon, or night.
- `check_in_time`: Trigger time for attendance buttons (HH:MM).
- `checkout_cooldown`: Hours to wait before prompting for check-out.
- `is_monitored`: Boolean toggle for attendance tracking.

### `students`
Mapping between Discord users and their local status.
- `discord_id`: User's Discord ID.
- `channel_id`: Private provisioned channel ID.
- `role_id`: Primary role (relation to `roles`).
- `shift`: Current shift.

### `attendances`
Logs for daily presence.
- `student_id`: Relation to `students`.
- `clock_in` / `clock_out`: Timestamps.
- `status`: pending_checkout, completed, absent, late, justified.
- `checkout_prompt_sent`: Flag to prevent duplicate buttons.

### `broadcasts`
Scheduled messaging tasks.
- `content`: Message body.
- `target_roles`: JSON list of roles to receive the message.
- `schedule_time`: When to send.
- `status`: scheduled, processing, completed, failed.

## Relationships
- **Student -> Role:** Each student belongs to a primary role that determines their schedule.
- **Attendance -> Student:** Tracks individual history.
- **Role -> Guild:** Roles are scoped to specific Discord servers.
