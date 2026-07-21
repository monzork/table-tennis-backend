package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestParseCreateEventCommand(t *testing.T) {
	app := fiber.New()
	app.Post("/create", func(c *fiber.Ctx) error {
		cmd, err := parseCreateEventCommand(c)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		return c.JSON(cmd)
	})

	t.Run("Valid Form Data", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Test Event")
		data.Set("type", "singles")
		data.Set("format", "elimination")
		data.Set("groupPassCount", "2")
		data.Add("participant_ids[]", "p1")
		data.Add("participant_ids[]", "p2")
		data.Add("new_player_first_name[]", "John")
		data.Add("new_player_last_name[]", "Doe")
		data.Add("new_player_gender[]", "M")
		data.Set("stage_rule[group][best_of]", "5")
		data.Set("stage_rule[group][points_to_win]", "11")
		data.Set("stage_rule[group][points_margin]", "2")
		data.Add("division_rule[division_id][]", "div1")
		data.Add("division_rule[division_id][]", "")
		data.Set("division_rule[div1][group][best_of]", "5")
		data.Set("division_formats[div1]", "elimination")
		data.Set("division_group_pass_counts[div1]", "2")
		data.Set("division_losers_group_pass_counts[div1]", "3")
		data.Set("division_group_counts[div1]", "4")
		data.Set("eventId", "parent-event-1")

		req := httptest.NewRequest("POST", "/create", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Invalid Body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/create", strings.NewReader("invalid body"))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode == 200 {
			t.Errorf("expected error, got 200")
		}
	})
}

func TestParseUpdateEventCommand(t *testing.T) {
	app := fiber.New()
	app.Post("/update/:id", func(c *fiber.Ctx) error {
		cmd, err := parseUpdateEventCommand(c)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		return c.JSON(cmd)
	})

	t.Run("Valid Form Data", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Updated Event")
		data.Set("type", "doubles")
		data.Set("registrationOpen", "true")
		data.Set("format", "elimination")
		data.Set("groupPassCount", "2")
		data.Add("participant_ids[]", "p1")
		data.Add("participant_ids[]", "p2")
		data.Add("new_player_first_name[]", "John")
		data.Add("new_player_last_name[]", "Doe")
		data.Add("new_player_gender[]", "M")
		data.Set("stage_rule[group][best_of]", "5")
		data.Set("stage_rule[group][points_to_win]", "11")
		data.Set("stage_rule[group][points_margin]", "2")
		data.Add("division_rule[division_id][]", "div1")
		data.Add("division_rule[division_id][]", "")
		data.Set("division_rule[div1][group][best_of]", "5")
		data.Set("division_formats[div1]", "elimination")
		data.Set("division_group_pass_counts[div1]", "2")
		data.Set("division_losers_group_pass_counts[div1]", "3")
		data.Set("division_group_counts[div1]", "4")
		data.Set("eventId", "parent-event-1")
		data.Set("skipElo", "on")

		req := httptest.NewRequest("POST", "/update/evt123", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Invalid Body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/update/evt123", strings.NewReader("invalid body"))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		if resp.StatusCode == 200 {
			t.Errorf("expected error, got 200")
		}
	})
}
