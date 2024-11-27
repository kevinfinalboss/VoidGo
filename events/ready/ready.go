package ready

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/internal/types"
)

var (
	startTime  = time.Now()
	activities = []string{
		"Helping users",
		"Processing commands",
		"Running maintenance",
	}
)

var ReadyEvent = &types.Event{
	Name: "ready",
	Handler: func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("Bot is ready! Logged in as %s#%s\n", s.State.User.Username, s.State.User.Discriminator)
		fmt.Printf("Bot ID: %s\n", s.State.User.ID)

		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Number of goroutines: %d\n", runtime.NumGoroutine())

		fmt.Printf("Connected to %d guilds\n", len(s.State.Guilds))

		go statusRotation(s)

		err := s.UpdateGameStatus(0, "Starting up...")
		if err != nil {
			fmt.Printf("Error setting presence: %v\n", err)
		}

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Printf("Memory Usage: %v MB\n", mem.Alloc/1024/1024)

		fmt.Printf("Startup completed in %v\n", time.Since(startTime))

		for _, guild := range s.State.Guilds {
			fmt.Printf("Guild: %s (ID: %s)\n", guild.Name, guild.ID)
			memberCount := guild.MemberCount
			fmt.Printf("- Members: %d\n", memberCount)
		}
	},
}

func statusRotation(s *discordgo.Session) {
	ticker := time.NewTicker(1 * time.Minute)
	currentActivity := 0

	for range ticker.C {
		status := activities[currentActivity]
		err := s.UpdateGameStatus(0, status)
		if err != nil {
			fmt.Printf("Error updating status: %v\n", err)
		}

		currentActivity = (currentActivity + 1) % len(activities)

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Printf("Current Memory Usage: %v MB\n", mem.Alloc/1024/1024)
	}
}

func AddCustomStatus(status string) {
	activities = append(activities, status)
}

func GetUptime() time.Duration {
	return time.Since(startTime)
}

func GetGuildCount(s *discordgo.Session) int {
	return len(s.State.Guilds)
}
