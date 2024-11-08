package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/matthewyuh246/notification2/models"
	"github.com/robfig/cron"
)

var events []models.Event
var c = cron.New()

func loadEnv() {
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Println(".env読み込みエラー: %v", err)
	}
	fmt.Println(".envを読み込みました。")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content[:4] == "!add" {
		parts := strings.TrimSpace(m.Content[5:])
		details := strings.SplitN(parts, "|", 2)

		if len(details) != 2 {
			s.ChannelMessageSend(m.ChannelID, "形式: !add <YYYY-MM-DD HH:MM>|<予定タイトル>")
			return
		}
		fmt.Println("Received date string:", details[0]) // デバッグ用出力

		eventTime, err := time.Parse("2006-01-02 15:04", details[0])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "日付形式が無効です。例: 2024-12-31 13:00")
			fmt.Println("Date parsing error:", err) // エラーの内容を出力
			return
		}

		event := models.Event{
			Title:     details[1],
			Time:      eventTime,
			ChannelID: m.ChannelID,
		}
		events = append(events, event)

		scheduleReminders(s, event)
		s.ChannelMessageSend(m.ChannelID, "予定を追加しました: "+details[1])
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s は %s です。", details[1], eventTime.Format("2006-01-02 15:04")))
	}
}

func scheduleReminders(s *discordgo.Session, event models.Event) {
	reminderTime36 := event.Time.Add(-36 * time.Hour)
	c.AddFunc(reminderTime36.Format("05 04 15 02 01 *"), func() {
		sendReminder(s, event.ChannelID, event.Title, 36)
	})

	reminderTime12 := event.Time.Add(-12 * time.Hour)
	c.AddFunc(reminderTime12.Format("05 04 15 02 01 *"), func() {
		sendReminder(s, event.ChannelID, event.Title, 12)
	})
}

func sendReminder(s *discordgo.Session, channelID, title string, hours int) {
	s.ChannelMessageSend(channelID, fmt.Sprintf("%sが%d時間後にあります！\n参加が厳しい人はdiscordかtwitterに連絡ください。", title, hours))
}

func main() {
	loadEnv()
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		fmt.Println("Botトークンが見つかりません。")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Discordのセッションでエラー, ", err)
		return
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("接続エラー", err)
		return
	}
	defer dg.Close()
	fmt.Println("Bot稼働中。CTRL+Cで終了。")

	c.Start()
	defer c.Stop()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
