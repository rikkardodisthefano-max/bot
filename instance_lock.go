package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	instanceLockFile = "bot-pro.instance.lock"
	lockStaleAfter   = 90 * time.Second
	lockHeartbeat    = 20 * time.Second
)

var releaseInstanceLock func()

func init() {
	unlock, err := acquireInstanceLock()
	if err != nil {
		log.Fatalf("🛑 Bot-Pro não iniciado: %v", err)
	}
	releaseInstanceLock = unlock

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		if releaseInstanceLock != nil {
			releaseInstanceLock()
		}
	}()
}

func acquireInstanceLock() (func(), error) {
	path := os.Getenv("BOT_PRO_LOCK_FILE")
	if strings.TrimSpace(path) == "" {
		path = instanceLockFile
	}

	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			pid := os.Getpid()
			_, _ = fmt.Fprintf(f, "pid=%d\nstarted=%s\n", pid, time.Now().UTC().Format(time.RFC3339))
			_ = f.Close()

			stop := make(chan struct{})
			var once sync.Once
			go func() {
				ticker := time.NewTicker(lockHeartbeat)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						_ = os.Chtimes(path, time.Now(), time.Now())
					case <-stop:
						return
					}
				}
			}()

			return func() {
				once.Do(func() {
					close(stop)
					_ = os.Remove(path)
				})
			}, nil
		}

		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		if time.Since(info.ModTime()) < lockStaleAfter {
			data, _ := os.ReadFile(path)
			return nil, fmt.Errorf("outra instância do Bot-Pro já está em execução (%s)", strings.TrimSpace(string(data)))
		}
		if removeErr := os.Remove(path); removeErr != nil {
			return nil, fmt.Errorf("lock antigo encontrado e não pôde ser removido: %w", removeErr)
		}
	}

	return nil, fmt.Errorf("não foi possível adquirir o lock de instância")
}

func lockProcessID() string {
	return strconv.Itoa(os.Getpid())
}
