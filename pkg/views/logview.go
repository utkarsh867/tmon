package views

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
)

type LogView struct {
  sub chan LogStreamMessage
  content string
  ready bool
  viewport viewport.Model
  pty ssh.Pty
}

type LogStreamMessage struct {
  content string
}

type LogStreamEndMessage struct {}


func CreateLogModel(pty ssh.Pty) LogView {
  content := ""
  return LogView {
    content: string(content),
    pty: pty,
    sub: make(chan LogStreamMessage),
  }
}

func (m LogView) Init() tea.Cmd {
  return tea.Batch(
    m.waitForNewStreamLog(),
    m.runStreamLog(),
  )
}

func (m LogView) View() string {
  if !m.ready {
    return "\n Initializing..."
  }

  return fmt.Sprintf("%s\n", m.viewport.View())
}

func (m LogView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var (
    cmd  tea.Cmd
    cmds []tea.Cmd
  )
  switch msg := msg.(type) {
  case tea.KeyMsg:
    if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
      return m, tea.Quit
    }
  case LogStreamMessage:
    newContent := msg.content
    m.content = m.content + newContent
    m.viewport.SetContent(m.content)
    m.viewport.GotoBottom()
  case tea.WindowSizeMsg:
    if !m.ready {
      m.viewport = viewport.New(msg.Width/3, msg.Height)
      m.viewport.YPosition = 0
      m.viewport.HighPerformanceRendering = false
      m.viewport.SetContent(m.content)
      m.ready = true
    } else {
      m.viewport.Width = m.pty.Window.Width / 3
      m.viewport.Height = m.pty.Window.Height
    }
  }

  m.viewport, cmd = m.viewport.Update(msg)
  cmds = append(cmds, cmd, m.waitForNewStreamLog())
  return m, tea.Batch(cmds...)
}

func (m LogView) runStreamLog() tea.Cmd {
  return func() tea.Msg {
    logs := exec.Command("dmesg", "-w")
    reader, err := logs.StdoutPipe()
    if err != nil {
      log.Fatal("Error")
    }

    err = logs.Start()
    if err != nil {
      log.Fatal("Error")
    }

    buf := make([]byte, 1024)
    for {
      n, err := reader.Read(buf)
      if err != nil {
        break
      }
      m.sub <- LogStreamMessage{content: string(buf[0:n])}
    }
    return LogStreamEndMessage{}
  }
}

func (m *LogView) waitForNewStreamLog() tea.Cmd {
  return func() tea.Msg {
    content := <- m.sub
    return content
  }
}
