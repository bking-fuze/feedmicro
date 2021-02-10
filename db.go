package main

import (
	"context"
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"time"
	"fmt"
)

var DB *sql.DB

func dbOpen(connectionString string) error {
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return err
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	DB = db;
	return nil
}

func dbClose() {
	DB.Close()
}

type meetingInstanceInfo struct {
	startedAt time.Time
	endedAt   time.Time
}

func dbGetMeetingInstanceInfo(ctx context.Context, id int64) (*meetingInstanceInfo, error) {
	var startedAt mysql.NullTime
	var endedAt mysql.NullTime
	err := DB.QueryRowContext(ctx, "SELECT started_at, ended_at FROM meeting_instances WHERE id=? and state='Ended'", id).Scan(&startedAt, &endedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if !startedAt.Valid {
		return nil, fmt.Errorf("null started_at for %d", id)
	} else if !endedAt.Valid {
		return nil, fmt.Errorf("null ended_at for %d", id)
	} else {
		return &meetingInstanceInfo{
			startedAt: startedAt.Time,
			endedAt: endedAt.Time,
		}, nil
	}
}

func dbDeviceExists(ctx context.Context, deviceId string) (bool, error) {
	var one int
	err := DB.QueryRowContext(ctx, "SELECT 1 FROM device WHERE id=?", deviceId).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func dbQueryForTime(ctx context.Context, query string, args ...interface{}) (*time.Time, error) {
	var t mysql.NullTime
	err := DB.QueryRowContext(ctx, query, args...).Scan(&t)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if !t.Valid {
		return nil, nil
	} else {
		return &t.Time, nil
	}
}

func dbDownloadTokenCreatedAt(ctx context.Context, downloadToken string) (*time.Time, error) {
	return dbQueryForTime(ctx, "SELECT created_at FROM launch_tokens WHERE token=?", downloadToken)
}

func dbMeetingInstanceStartedAt(ctx context.Context, id int64) (*time.Time, error) {
	return dbQueryForTime(ctx, "SELECT started_at FROM meeting_instances WHERE id=?", id)
}
