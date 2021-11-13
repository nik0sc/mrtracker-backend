package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/smrt"
)

const (
	pollInterval = 30 * time.Second
	pollTimeout  = 30 * time.Second
	timeRound    = 30
	logName      = "recorder.log"
	envDsn       = "DSN"
	saveCommand  = "insert into recorded_position (day_of_week, seconds_of_day, time, name, line_repr) values (?,?,?,?,?)"
)

func main() {
	// create name list from line data
	names := data.GetNames()

	// connect to recorder db
	dsn := os.Getenv(envDsn)
	if dsn == "" {
		panic("where is dsn?")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(1 * time.Hour)
	defer func() {
		err = db.Close()
		if err != nil {
			log.Printf("error closing db: %v", err)
		}
	}()

	// rotate old log file
	logStat, err := os.Stat(logName)
	if err == nil {
		newname := fmt.Sprintf("%s.%s", logName, logStat.ModTime().Format("060102.150405"))
		err = os.Rename(logName, newname)
		if err != nil {
			panic(err)
		}
	} else if !os.IsNotExist(err) {
		panic(err)
	}

	f, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// round time to a nice number - multiple of 30s
	delta := timeRound - (time.Now().Second() % timeRound)
	<-time.After(time.Duration(delta) * time.Second)
	tick := time.Tick(pollInterval)
	fmt.Println("delay over, starting...")

	running := true
	for running {
		err = func() error {
			pollCtx, pollCancel := context.WithTimeout(ctx, pollTimeout)
			defer pollCancel()

			now := time.Now()

			results, _, err := smrt.GetN(pollCtx, 0, 10, names...)
			if err != nil {
				return err
			}

			lines := map[string]string{
				"ns1": smrt.ToModel(results, data.NS_1).ToPosition().ToString(),
				"ns2": smrt.ToModel(results, data.NS_2).ToPosition().ToString(),
				"ew1": smrt.ToModel(results, data.EW_1).ToPosition().ToString(),
				"ew2": smrt.ToModel(results, data.EW_2).ToPosition().ToString(),
				"cg1": smrt.ToModel(results, data.CG_1).ToPosition().ToString(),
				"cg2": smrt.ToModel(results, data.CG_2).ToPosition().ToString(),
			}

			err = save(pollCtx, db, now, lines)
			if err != nil {
				return err
			}
			return nil
		}()

		if err != nil {
			log.Printf("error: %v", err)
		}

		select {
		case <-ctx.Done():
			running = false
		case <-tick:
		}
	}
}

func save(ctx context.Context, db *sql.DB, now time.Time, lines map[string]string) error {
	stmt, err := db.PrepareContext(ctx, saveCommand)
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		_ = stmt.Close()
	}(stmt)

	dayOfWeek := int(now.Weekday())
	secondsOfDay := now.Hour()*3600 + now.Minute()*60 + now.Second()

	for name, linerepr := range lines {
		_, err := stmt.ExecContext(ctx, dayOfWeek, secondsOfDay, now, name, linerepr)
		if err != nil {
			return err
		}
	}
	return nil
}
