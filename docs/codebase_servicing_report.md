# Codebase Servicing Report: Background Processes

## Overview
Chantry relies on several background workers running within the Go Daemon to automate Discord community management.

## 1. Attendance Scheduler (`StartDynamicCron`)
- **Frequency:** Every 1 minute.
- **Logic:** 
    1. Queries the `roles` collection for any role where `check_in_time` matches the current system time.
    2. Identifies all active students mapped to those roles.
    3. Sends interactive "Check-in" buttons to each student's private `channel_id`.
- **Timezone:** Respects the `TIMEZONE` env variable to ensure buttons appear at the correct local time.

## 2. Clock-Out Ticker (`StartClockOutTicker`)
- **Frequency:** Every 1 minute.
- **Logic:**
    1. Finds attendance records with the status `pending_checkout`.
    2. Calculates if the time since `clock_in` exceeds the role's `checkout_cooldown`.
    3. If exceeded, dispatches a "Check-out" button to the student's channel and flags the record to prevent re-sending.

## 3. Broadcast Worker (`StartBroadcastWorker`)
- **Logic:**
    1. Continuously polls the `broadcasts` collection for records with `status="scheduled"` and `schedule_time <= now`.
    2. Marks the record as `processing`.
    3. Iterates through the target roles, fetches members, and sends messages via Discord.
    4. Updates metrics (sent vs. errors) and marks as `completed`.

## 4. Provisioning Engine (`ProvisionUsecase`)
- **Type:** On-demand (triggered by UI).
- **Features:**
    - **Channel Creation:** Automatically creates private text channels with strict permission overrides (only the student and managers can see).
    - **Healing:** A "Healing" logic that repairs missing channels or incorrect permissions for existing students without duplicating work.
    - **Naming Convention:** Standardizes channel names based on student nicknames for easy navigation.
