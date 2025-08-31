package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/uptrace/bun"
)

const backupDir = "./backups"
const maxBackups = 7

func StartBackupScheduler(ctx context.Context, db *bun.DB) (gocron.Scheduler, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create backup dir: %w", err)
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	_, err = s.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(2, 0, 0))),
		gocron.NewTask(func() {
			if err := createBackup(ctx, db); err != nil {
				fmt.Println("Error creating backup:", err)
				return
			}
			if err := rotateBackups(); err != nil {
				fmt.Println("Error rotating backups:", err)
			}
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup job: %w", err)
	}

	s.Start()
	return s, nil
}

func createBackup(ctx context.Context, db *bun.DB) error {
	timestamp := time.Now().Format("20060102_150405")
	dbPath := filepath.Join(backupDir, fmt.Sprintf("backup_%s.db", timestamp))
	gzPath := dbPath + ".gz"

	_, err := db.NewRaw(fmt.Sprintf("VACUUM INTO '%s'", dbPath)).Exec(ctx)
	if err != nil {
		return err
	}

	if err := compressFile(dbPath, gzPath); err != nil {
		return fmt.Errorf("failed to compress backup: %w", err)
	}

	if err := os.Remove(dbPath); err != nil {
		fmt.Println("Failed to remove uncompressed file:", dbPath, err)
	}

	fmt.Println("Backup created & compressed:", gzPath)
	return nil
}

func compressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	writer := gzip.NewWriter(out)
	defer writer.Close()

	_, err = io.Copy(writer, in)
	return err
}

func rotateBackups() error {
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	var backups []fs.DirEntry
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "backup_") && strings.HasSuffix(f.Name(), ".db.gz") {
			backups = append(backups, f)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		fi, _ := backups[i].Info()
		fj, _ := backups[j].Info()
		return fi.ModTime().After(fj.ModTime())
	})

	if len(backups) > maxBackups {
		for _, f := range backups[maxBackups:] {
			path := filepath.Join(backupDir, f.Name())
			if err := os.Remove(path); err != nil {
				fmt.Println("Failed to remove old backup:", path, err)
			} else {
				fmt.Println("Removed old backup:", path)
			}
		}
	}

	return nil
}
