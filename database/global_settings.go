package database

import "time"

func SaveGlobalActivity(activityType int, activity string) error {
	_, err := DB.Exec(`INSERT INTO bot_settings (guild_id, activity_type, activity) VALUES ('__global__', ?, ?) ON CONFLICT(guild_id) DO UPDATE SET activity_type=excluded.activity_type, activity=excluded.activity`, activityType, activity)
	return err
}

func GetGlobalActivity() (int, string) {
	if DB == nil { return 0, "" }
	var kind int
	var activity string
	if err := DB.QueryRow(`SELECT activity_type, activity FROM bot_settings WHERE guild_id='__global__'`).Scan(&kind, &activity); err != nil { return 0, "" }
	return kind, activity
}

func CurrentUTCMinute() int64 { return time.Now().UTC().Truncate(time.Minute).Unix() }
