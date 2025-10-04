package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cybellereaper/wizturtle/v2/internal/wizserver"
)

func main() {
	var (
		gameDir      string
		loginAddr    string
		gameAddr     string
		maxPlayers   uint
		zoneCount    uint
		zoneCapacity uint
	)

	flag.StringVar(&gameDir, "game-dir", "", "Path to the Wizard101 game data directory")
	flag.StringVar(&loginAddr, "login-addr", "127.0.0.1:12500", "Address to bind the login service")
	flag.StringVar(&gameAddr, "game-addr", "127.0.0.1:12501", "Address to bind the game service")
	flag.UintVar(&maxPlayers, "max-players", 100, "Maximum number of concurrent players in the realm")
	flag.UintVar(&zoneCount, "zones", 50, "Number of zones to create in the realm")
	flag.UintVar(&zoneCapacity, "zone-capacity", 10, "Maximum number of players per zone")
	flag.Parse()

	if gameDir == "" {
		fmt.Fprintln(os.Stderr, "--game-dir must be provided")
		os.Exit(2)
	}

	if _, err := os.Stat(gameDir); err != nil {
		fmt.Fprintf(os.Stderr, "game directory validation failed: %v\n", err)
		os.Exit(2)
	}

	absGameDir, err := filepath.Abs(gameDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve game directory: %v\n", err)
		os.Exit(2)
	}

	log.Printf("Starting WizTurtle server with data directory %s", absGameDir)

	realm := wizserver.NewRealm(uint32(maxPlayers), uint32(zoneCount), uint32(zoneCapacity))
	log.Printf("Realm initialised with %d zones", len(realm.Zones()))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	loginSrv, err := startListener(ctx, &wg, loginAddr, "login")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer loginSrv()

	gameSrv, err := startListener(ctx, &wg, gameAddr, "game")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer gameSrv()

	<-ctx.Done()
	log.Println("Shutting down... waiting for active connections to drain")
	wg.Wait()
	log.Println("Server stopped cleanly")
}

type stopFunc func()

func startListener(ctx context.Context, wg *sync.WaitGroup, addr, name string) (stopFunc, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					if ne, ok := err.(net.Error); ok && ne.Temporary() {
						log.Printf("temporary %s accept error: %v", name, err)
						time.Sleep(100 * time.Millisecond)
						continue
					}
					log.Printf("stopping %s listener: %v", name, err)
					return
				}
			}

			wg.Add(1)
			go handleConnection(ctx, wg, conn, name)
		}
	}()

	return func() {
		cancel()
		_ = listener.Close()
	}, nil
}

func handleConnection(ctx context.Context, wg *sync.WaitGroup, conn net.Conn, name string) {
	defer wg.Done()
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("failed to set deadline for %s connection: %v", name, err)
		return
	}

	greeting := fmt.Sprintf("Welcome to the WizTurtle %s service!\n", name)
	if _, err := conn.Write([]byte(greeting)); err != nil {
		log.Printf("failed to write greeting to %s client: %v", name, err)
		return
	}

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		if !errors.Is(err, os.ErrDeadlineExceeded) && !errors.Is(err, io.EOF) {
			log.Printf("read error on %s connection: %v", name, err)
		}
		return
	}

	response := fmt.Sprintf("You said: %s", strings.TrimSpace(string(buf[:n])))
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("failed to send response to %s client: %v", name, err)
	}

	select {
	case <-ctx.Done():
	default:
		// keep the connection open briefly to emulate activity
		time.Sleep(250 * time.Millisecond)
	}
}
