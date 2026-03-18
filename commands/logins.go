package commands

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type sessionInfo struct {
	User     string
	TTY      string
	LoginAt  time.Time
	Method   string // "SSH" або "Local"
	FromHost string
}

func LoggedInUsersCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	list, err := listViaLoginctl()
	if err != nil || len(list) == 0 {
		// fallback, якщо loginctl нічого не дав
		list, err = listViaWhoW()
	}
	if err != nil {
		replyError(bot, msg, err, "Не вдалося отримати активні сесії")
		return
	}
	if len(list) == 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Немає активних сесій."))
		return
	}

	var b bytes.Buffer
	b.WriteString("👥 *Активні користувачі*\n")
	for _, s := range list {
		uptime := "-"
		if !s.LoginAt.IsZero() {
			uptime = time.Since(s.LoginAt).Round(time.Second).String()
		}
		line := fmt.Sprintf(
			"— *%s* • %s\n   ⌨️ TTY: `%s`\n   🕒 Вхід: %s (UTC)\n   (%s тому)",
			s.User, s.Method, s.TTY, humanTime(s.LoginAt), uptime,
		)
		if s.FromHost != "" {
			line += fmt.Sprintf("\n   🌐 IP: %s", s.FromHost)
		}
		b.WriteString(line + "\n")
	}

	out := tgbotapi.NewMessage(msg.Chat.ID, b.String())
	out.ParseMode = "Markdown"
	bot.Send(out)
}

func listViaLoginctl() ([]sessionInfo, error) {
	out, err := exec.Command("loginctl", "list-sessions", "--no-legend", "--no-pager").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return nil, nil
	}

	var res []sessionInfo
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// формат: <ID> <USER> <SEAT> <TTY>
		fs := strings.Fields(ln)
		if len(fs) < 4 {
			continue
		}
		id, user, tty := fs[0], fs[1], fs[3]

		props, err := exec.Command("loginctl", "show-session", id).Output()
		if err != nil {
			continue
		}
		p := toPropMap(string(props))

		method := "Local"
		if p["Remote"] == "yes" {
			method = "SSH"
		}
		from := p["RemoteHost"]
		loginAt := parseEpoch(p["TimestampEpoch"])

		res = append(res, sessionInfo{
			User:     user,
			TTY:      tty,
			LoginAt:  loginAt,
			Method:   method,
			FromHost: from,
		})
	}
	return res, nil
}

func listViaWhoW() ([]sessionInfo, error) {
	// who -u (у різних дистрибутивах формат трохи різниться)
	cmd := exec.Command("who", "-u")
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var res []sessionInfo
	// Дістанемо час логіна з початку рядка (різні варіанти дат)
	reTs := regexp.MustCompile(`^\S+\s+\S+\s+(\d{4}-\d{2}-\d{2}|\w+\s+\d+)\s+(\d{2}:\d{2})`)

	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		fs := strings.Fields(ln)
		if len(fs) < 2 {
			continue
		}
		user := fs[0]
		tty := fs[1]

		// Час логіну
		loginAt := time.Time{}
		if m := reTs.FindStringSubmatch(ln); len(m) == 3 {
			for _, layout := range []string{
				"2006-01-02 15:04",
				"Jan _2 15:04",
				"Mon Jan _2 15:04",
			} {
				if t, e := time.Parse(layout, m[1]+" "+m[2]); e == nil {
					if t.Year() == 0 {
						now := time.Now()
						t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
					}
					loginAt = t
					break
				}
			}
		}

		// host/IP — зазвичай останнє поле; (:0) означає локальну X‑сесію
		from := ""
		if len(fs) >= 5 {
			from = fs[len(fs)-1]
			if strings.HasPrefix(from, "(") && strings.HasSuffix(from, ")") {
				from = strings.Trim(from, "()")
			}
			if from == ":" || strings.HasPrefix(from, ":") {
				from = ""
			}
		}
		method := "Local"
		if from != "" && from != "." {
			method = "SSH"
		}

		res = append(res, sessionInfo{
			User:     user,
			TTY:      tty,
			LoginAt:  loginAt,
			Method:   method,
			FromHost: from,
		})
	}
	return res, nil
}

func toPropMap(s string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		if i := strings.IndexByte(line, '='); i > 0 {
			m[line[:i]] = line[i+1:]
		}
	}
	return m
}

func parseEpoch(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// loginctl зазвичай дає мікросекунди з епохи
	if len(s) >= 16 {
		var us int64
		fmt.Sscan(s, &us)
		sec := us / 1_000_000
		nsec := (us % 1_000_000) * 1000
		return time.Unix(sec, nsec)
	}
	var sec int64
	fmt.Sscan(s, &sec)
	return time.Unix(sec, 0)
}

func humanTime(t time.Time) string {
	if t.IsZero() {
		return "н/д"
	}
	return t.Format("2006-01-02 15:04:05")
}

func replyError(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, err error, prefix string) {
	log.Println(prefix+":", err)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ %s: %v", prefix, err)))
}
