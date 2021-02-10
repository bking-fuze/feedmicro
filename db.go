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
