package main

import (
	"fmt"
	"io/ioutil"
	"os"
	//"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/nlopes/slack"
	"github.com/spf13/cobra"
)

const (
	FlagStandUpDir = "stand-ups"
	FlagToken      = "token"
	FlagChannel    = "channel"
	FlagName       = "name"
)

type StandUp struct {
	client *slack.Client
	rtm    *slack.RTM
	id     string

	channel string
	name    string
	token   string
	dir     string
}

var RootCmd = &cobra.Command{
	Use:   "stand-up",
	Short: "Post your stand-up to slack.",
	Run: func(cmd *cobra.Command, args []string) {

		day := time.Now().Weekday()

		if day == time.Saturday || day == time.Sunday {
			Error("You're not working today!\n")
		}

		token, err := cmd.PersistentFlags().GetString(FlagToken)
		if err != nil {
			Errorf("failed to get slack token flag: %v", err)
		}
		if token == "" {
			Error("Slack token not set, exiting.\n")
		}

		name, err := cmd.PersistentFlags().GetString(FlagName)
		if err != nil {
			Errorf("failed to get client name flag: %v", err)
		}

		channel, err := cmd.PersistentFlags().GetString(FlagChannel)
		if err != nil {
			Errorf("failed to get stand-up channel flag: %v", err)
		}

		s := &StandUp{
			client:  slack.New(token),
			token:   token,
			name:    name,
			channel: channel,
		}

		path, err := cmd.PersistentFlags().GetString(FlagStandUpDir)
		if err != nil {
			s.Must(fmt.Errorf("failed to get stand-up directory flag: %v", err))
		}
		path, err = homedir.Expand(path)
		if err != nil {
			s.Must(fmt.Errorf("failed to expand stand-up directory: %v", err))
		}
		s.dir = path

		standup, err := s.CreateStandUp()
		s.Must(err)

		rtm := s.client.NewRTM()
		users, err := rtm.GetUsers()
		for _, u := range users {
			if u.Name == s.name {
				s.id = u.ID
				break
			}
		}
		if s.id == "" {
			s.Must(fmt.Errorf("failed to find user: %s", s.name))
		}
		s.rtm = rtm

		s.Must(s.SendStandUpMessage(standup))
	},
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		Errorf("%v\n", err)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP(FlagStandUpDir, "f", "/home/josh/Jetstack/standups", "Set directory of standups")
	RootCmd.PersistentFlags().StringP(FlagToken, "t", "", "Set you client slack token")
	RootCmd.PersistentFlags().StringP(FlagChannel, "c", "stand-ups", "Set channel to post stand-up")
	RootCmd.PersistentFlags().StringP(FlagName, "n", "josh.van.leeuwen", "Set name of slack client")
}

func (s *StandUp) Must(err error) {
	if err != nil {
		if s.rtm != nil {
			params := slack.NewPostMessageParameters()
			params.AsUser = false
			errStr := fmt.Sprintf("An error occured when trying to post your stand up!\n%v\n", err)
			_, _, err := s.rtm.PostMessage(s.id, errStr, params)
			if err != nil {
				fmt.Fprintf(os.Stderr, "an error occured sending error to slack client: %v\n", err)
			}
		}

		Errorf("%v\n", err)
	}
}

func Error(err string) {
	fmt.Fprint(os.Stderr, err)
	os.Exit(1)
}

func Errorf(err string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, err, a)
	os.Exit(1)
}

func (s *StandUp) SendStandUpMessage(standup string) error {
	params := slack.NewPostMessageParameters()
	params.AsUser = true
	respChanel, respTimestamp, err := s.client.PostMessage(s.channel, standup, params)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n%s\n", respChanel, respTimestamp)
	return nil
}

func (s *StandUp) CreateStandUp() (string, error) {
	now := time.Now()
	prevDay := now.Add(-time.Hour * 24)
	for prevDay.Weekday() == time.Saturday || prevDay.Weekday() == time.Sunday {
		prevDay = prevDay.Add(-time.Hour * 24)
	}

	todayPath := s.createPath(now)
	prevPath := s.createPath(prevDay)

	if _, err := os.Stat(prevPath); err != nil && os.IsNotExist(err) {
		s.Must(s.VimStandUp(prevPath, "Enter previous stand-up"))
	}

	s1, err := ioutil.ReadFile(prevPath)
	if err != nil {
		return "", fmt.Errorf("failed to read last stand-up: %v", err)
	}

	if _, err = os.Stat(todayPath); err != nil && os.IsNotExist(err) {
		s.Must(s.VimStandUp(todayPath, "Enter todays stand-up"))
	}

	s2, err := ioutil.ReadFile(todayPath)
	if err != nil {
		return "", fmt.Errorf("failed to read today's stand-up: %v", err)
	}

	standup := s.generateStandUp(s1, s2, now.Weekday().String(), prevDay.Weekday().String())

	return standup, nil
}

func (s *StandUp) VimStandUp(path, message string) error {
	//err := ioutil.WriteFile(path, []byte(message), 0644)
	//if err != nil {
	//	return fmt.Errorf("failed to write to next stand up file: %v", err)
	//}

	//cmd := exec.Command("termite", fmt.Sprintf("--exec=vim %s", path))
	//return cmd.Run()
	return nil
}

func (s *StandUp) generateStandUp(s1, s2 []byte, today, prevDay string) string {
	return fmt.Sprintf("```\n%s:\n%s\n%s:\n%s```", prevDay, s1, today, s2)
}

func (s *StandUp) createPath(t time.Time) string {
	day := strconv.Itoa(t.Day())
	if len(day) < 2 {
		day = fmt.Sprintf("0%s", day)
	}

	month := strconv.Itoa(int(t.Month()))
	if len(month) < 2 {
		month = fmt.Sprintf("0%s", month)
	}

	path := fmt.Sprintf("%s_%s_%d", day, month, (t.Year() % 100))
	path = filepath.Join(s.dir, path)

	return path
}
