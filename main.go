package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	wlog "github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/utkarsh867/tmon/pkg/views"
	"github.com/muesli/termenv"
)

type MainView struct {
  systemDView tea.Model
  rightLog tea.Model
  pty ssh.Pty
}

func (m MainView) Init() tea.Cmd {
  return tea.Batch(
    m.systemDView.Init(),
    m.rightLog.Init(),
  )
}

func (m MainView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var (
    cmds []tea.Cmd
  )
  switch msg := msg.(type) {
  case tea.KeyMsg:
    if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
      return m, tea.Quit
    }
  default:
    var (
      cmd1 tea.Cmd
      cmd2 tea.Cmd
    )
    m.systemDView, cmd1 = m.systemDView.Update(msg)
    m.rightLog, cmd2 = m.rightLog.Update(msg)
    cmds = append(cmds, cmd1, cmd2)
  }
  return m, tea.Batch(cmds...)
}

func (m MainView) View() string {
  return lipgloss.JoinHorizontal(
    lipgloss.Left,
    m.systemDView.View(),
    m.rightLog.View(),
  )
}

func main() {
  f, err := tea.LogToFile("debug.log", "debug")
  if err != nil {
		log.Println("fatal:", err)
		os.Exit(1)
	}
  defer f.Close()

  p := func(s ssh.Session) *tea.Program {
    pty, _, active := s.Pty()
		if !active {
			wish.Fatalln(s, "no active terminal, skipping")
			return nil
		}

    initialModel := MainView{
      systemDView: views.CreateSystemDModel(pty),
      rightLog: views.CreateLogModel(pty),
      pty: pty,
    } 
    prog := tea.NewProgram(initialModel, append(bm.MakeOptions(s), tea.WithAltScreen(), tea.WithMouseCellMotion())...)

    return prog
  }

  s, err := wish.NewServer(
    wish.WithAddress(fmt.Sprintf("%s:%d", "localhost", 23234)),
    wish.WithMiddleware(
      bm.MiddlewareWithProgramHandler(p, termenv.ANSI256),
    ),
  )

  done := make(chan os.Signal, 1)
  signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
  wlog.Info("Starting SSH server", "host", "localhost", "port", 23234)

  go func() {
    if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed){
      wlog.Error("Could not start server", "error", err)
      done <- nil
    }
  }()
  <-done

  wlog.Info("Stopping SSH server")
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer func() { cancel() }()
  if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		wlog.Error("could not stop server", "error", err)
	}

}
