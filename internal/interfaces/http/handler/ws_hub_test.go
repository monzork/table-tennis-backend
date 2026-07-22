package handler_test

import (
	"sync"
	"testing"
	"time"

	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
)

func TestBracketHub_RegisterAndUnregister(t *testing.T) {
	if handler.GlobalBracketHub == nil {
		t.Fatal("GlobalBracketHub should be initialized")
	}

	dummyConn1 := &fiberws.Conn{}
	dummyConn2 := &fiberws.Conn{}
	tID := "tournament-123"

	handler.GlobalBracketHub.Register(tID, dummyConn1)
	handler.GlobalBracketHub.Register(tID, dummyConn2)
	handler.GlobalBracketHub.Unregister(tID, dummyConn1)
	handler.GlobalBracketHub.Unregister(tID, dummyConn2)
	handler.GlobalBracketHub.Unregister("non-existent", dummyConn1)
}

func TestBracketHub_BroadcastToEmpty(t *testing.T) {
	handler.GlobalBracketHub.Broadcast("non-existent-tID", map[string]string{"type": "reload"})
	handler.GlobalBracketHub.BroadcastHTML("non-existent-tID", "<div>updated</div>")
}

func TestBracketHub_ConcurrentAccess(t *testing.T) {
	hub := handler.GlobalBracketHub
	tID := "concurrent-tID"

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn := &fiberws.Conn{}
			hub.Register(tID, conn)
			hub.Broadcast(tID, map[string]string{"status": "testing"})
			hub.BroadcastHTML(tID, "<div>test</div>")
			hub.Unregister(tID, conn)
		}()
	}

	wg.Wait()
}

func TestWsBracketHandler(t *testing.T) {
	app := fiber.New()
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/brackets/:tournamentId", fiberws.New(handler.WsBracketHandler))

	go func() {
		app.Listen(":8081")
	}()
	time.Sleep(100 * time.Millisecond)

	url := "ws://localhost:8081/ws/brackets/test-tID"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer conn.Close()

	// Give it a moment to register
	time.Sleep(50 * time.Millisecond)

	// Broadcast should now hit this connection
	handler.GlobalBracketHub.Broadcast("test-tID", map[string]string{"msg": "hello"})

	// Read the message
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(msg) != `{"msg":"hello"}` {
		t.Errorf("Unexpected message: %s", string(msg))
	}

	// Close connection to trigger unregister
	conn.Close()
	time.Sleep(50 * time.Millisecond)

	// Stop the app
	app.Shutdown()
}
