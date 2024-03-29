package views

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Service struct {
  name string
  command string
  status string
}

type SystemDModel struct {
  services []Service
  list list.Model
  pty ssh.Pty
}

type ServiceUpdateMsg struct {}

func (i Service) Title() string {
  return i.name
}

func (i Service) Description() string {
  return i.status
}

func (i Service) FilterValue() string {
  return i.name
}

func CreateSystemDModel(pty ssh.Pty) SystemDModel {
  services := []Service {
    {
      name: "livestream",
      command: "livestream.service",
      status: "unknown",
    },
    {
      name: "wireguard",
      command: "wg-quick@wg0.service",
      status: "unknown",
    },
  }

  items := make([]list.Item, len(services))
  for i := 0; i < len(services); i++ {
    items[i] = services[i]
  }

  return SystemDModel {
    pty: pty,
    list: list.New(items, list.NewDefaultDelegate(), 0, 0),
    services: services,
  }
}

func (m SystemDModel) updateServiceStatus() tea.Cmd {
  return func() tea.Msg {
    for i := 0; i < len(m.services); i++ {
      srv := &m.services[i]
      cmd := exec.Command("systemctl", "check", srv.command)
      out, err := cmd.CombinedOutput()
      if err != nil {
        fmt.Printf("Error doing systemctl check %s\t%s\n", srv.command, err.Error())
        srv.status = "error"
      } else {
        srv.status = string(out)
      }
      fmt.Printf("Set status of service %s to %s\n", srv.name, srv.status)
    }
    return ServiceUpdateMsg{}
  }
}

func (m SystemDModel) Init() tea.Cmd {
  return tea.Batch(
    m.updateServiceStatus(),
  )
}

func (m SystemDModel) View() string {
  return docStyle.Render(m.list.View())
}

func (m SystemDModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var (
    cmd tea.Cmd
    cmds []tea.Cmd
  )
  switch msg := msg.(type) {
  case tea.WindowSizeMsg:
    h, v := docStyle.GetFrameSize()
    m.list.SetSize(msg.Width/3 - h, msg.Height - v)
  case ServiceUpdateMsg:
    for i := 0; i < len(m.services); i++ {
      fmt.Printf("Setting list item: %s\n", m.services[i])
      cmd = m.list.SetItem(i, m.services[i])
      cmds = append(cmds, cmd)
    }
  }
  m.list, cmd = m.list.Update(msg)
  cmds = append(cmds, cmd, m.updateServiceStatus())
  return m, tea.Batch(cmds...)
}

