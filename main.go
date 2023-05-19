package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhowden/tag"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type track struct {
	title       string
	description string
}

func (t track) Title() string       { return t.title }
func (t track) FilterValue() string { return t.title }
func (t track) Description() string { return t.description }

type audiofile struct {
	file        *os.File
	filename    string
	audioformat musicType
}

type (
	chgVol  int
	seekPos int
)

type audioKeyMap struct {
	Play         key.Binding
	TogglePause  key.Binding
	ToggleRepeat key.Binding
	SeekRight    key.Binding
	SeekLeft     key.Binding
	Delete       key.Binding
	MoveUp       key.Binding
	MoveDown     key.Binding
	VolumeUp     key.Binding
	VolumeDown   key.Binding
}

type model struct {
	list         list.Model
	err          error
	keys         *audioKeyMap
	audioContext *audio.Context
	progress     progress.Model
	repeat       bool
	volumechg    int
	seekchg      int
	initWinWidth int
}

type tickMsg time.Time

type mouseBtnPress int

const (
	mouseBtnLeft mouseBtnPress = iota
	mouseBtnRight
	mouseBtnMiddle
	mouseBtnNone
)

var (
	player          *Player
	fileList        []audiofile
	playstatus      string
	wasMousePressed mouseBtnPress
	progressStr     string
	currIdx         int
)

func newAudioKeyMap() *audioKeyMap {
	return &audioKeyMap{
		Play: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select & play"),
		),
		TogglePause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle pause"),
		),
		ToggleRepeat: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "toggle repeat"),
		),
		SeekRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "seek right"),
		),
		SeekLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "seek left"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "move up"),
		),
		MoveDown: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "move later"),
		),
		VolumeUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "increase volume"),
		),
		VolumeDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "decrease volume"),
		),
	}
}

func newModel(trackList []list.Item) *model {
	listKeys := newAudioKeyMap()
	newmodel := model{keys: listKeys}

	newmodel.list = list.New(trackList, list.NewDefaultDelegate(), 0, 0)
	newmodel.list.Title = "Tracks"

	// I needed to change default behavior as otherwise we would need to use
	// unintuitive keybindings
	newmodel.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "down"),
	)
	newmodel.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "up"),
	)
	newmodel.list.KeyMap.PrevPage = key.NewBinding(
		key.WithKeys("h", "pgup"),
		key.WithHelp("h/pgup", "prev page"),
	)
	newmodel.list.KeyMap.NextPage = key.NewBinding(
		key.WithKeys("l", "pgdown"),
		key.WithHelp("h/pgdn", "next page"),
	)
	newmodel.list.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.Play,
			listKeys.TogglePause,
			listKeys.ToggleRepeat,
			listKeys.SeekRight,
			listKeys.SeekLeft,
			listKeys.Delete,
			listKeys.MoveUp,
			listKeys.MoveDown,
			listKeys.VolumeUp,
			listKeys.VolumeDown,
		}
	}

	newmodel.progress = progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	playstatus = "  "

	newmodel.audioContext = audio.NewContext(sampleRate)
	return &newmodel
}

func (m model) Init() tea.Cmd {
	return m.tickUpd(0.0)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.initWinWidth = msg.Width - 2 - len(playstatus)
		m.progress.Width = m.initWinWidth
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, list.DefaultKeyMap().Quit):
			if player != nil {
				player.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Play):
			currIdx = m.list.Index()

			if player != nil {
				player.Rewind(currIdx)
			}

			var err error
			player, err = NewPlayer(fileList[currIdx].file, m.audioContext, fileList[currIdx].audioformat)
			if err != nil {
				m.list.NewStatusMessage("Cannot Play Audio: " + err.Error())
				progressStr = "--/--"
				m.progress.Width = m.initWinWidth - len(progressStr)
			} else {
				playstatus = "  "
				m.list.NewStatusMessage("Currently Playing: " + m.list.SelectedItem().(track).title)
			}

			return m, nil

		case key.Matches(msg, m.keys.VolumeUp):
			if player != nil {
				m.volumechg++
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return chgVol(m.volumechg)
				})
			}
			return m, nil

		case key.Matches(msg, m.keys.VolumeDown):
			if player != nil {
				m.volumechg--
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return chgVol(m.volumechg)
				})
			}
			return m, nil

		case key.Matches(msg, m.keys.SeekRight):
			if player != nil {
				m.seekchg++
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return seekPos(m.seekchg)
				})
			}
			return m, nil

		case key.Matches(msg, m.keys.SeekLeft):
			if player != nil {
				m.seekchg--
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return seekPos(m.seekchg)
				})
			}
			return m, nil

		case key.Matches(msg, m.keys.TogglePause):
			if player != nil {
				playstatus = player.TogglePause()
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleRepeat):
			m.repeat = !m.repeat
			return m, nil
		}

	case tea.MouseMsg:
		mouseEvent := tea.MouseEvent(msg)
		switch mouseEvent.Type {
		case tea.MouseLeft:
			wasMousePressed = mouseBtnLeft
		case tea.MouseRelease:
			switch wasMousePressed {
			case mouseBtnLeft:
				if player != nil && mouseEvent.Y == 0 && mouseEvent.X >= 1 && mouseEvent.X <= m.progress.Width {
					player.SeekTo(float64(mouseEvent.X-1) / float64(m.progress.Width))
					return m, nil
				}
				if player != nil && mouseEvent.Y == 0 && mouseEvent.X > m.progress.Width {
					playstatus = player.TogglePause()
					return m, nil
				}
			}
			wasMousePressed = mouseBtnNone
		}

	case tickMsg:
		fracDone := 0.0
		if player != nil {
			progressStr, fracDone = player.Update()
			m.progress.Width = m.initWinWidth - len(progressStr)
		}
		cmd := m.progress.SetPercent(fracDone)
		return m, tea.Batch(m.tickUpd(fracDone), cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case chgVol:
		if int(msg) == m.volumechg {
			player.SetVolume(m.volumechg)
			m.volumechg = 0
			return m, nil
		}

	case seekPos:
		if int(msg) == m.seekchg {
			player.Seek(m.seekchg)
			m.seekchg = 0
			return m, nil
		}

	default:
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) tickUpd(fracDone float64) tea.Cmd {
	if fracDone == 1.0 {
		m.nextSong()
	}
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) nextSong() {
	if player != nil {
		player.Rewind(currIdx)
	}

	if !m.repeat {
		currIdx++
		currIdx %= len(fileList)
	}

	var err error
	player, err = NewPlayer(fileList[currIdx].file, m.audioContext, fileList[currIdx].audioformat)

	if err != nil {
		m.list.NewStatusMessage("Cannot Play Audio: " + err.Error())
		progressStr = "--/--"
		m.progress.Width = m.initWinWidth - len(progressStr)
		if !m.repeat {
			m.nextSong()
		}
		return
	}

	m.list.Select(currIdx)
	playstatus = "  "
	m.list.NewStatusMessage("Currently Playing: " + m.list.SelectedItem().(track).title)
}

func (m model) View() string {
	progressive := " " + m.progress.View() + playstatus + progressStr + " \n"
	return lipgloss.JoinVertical(lipgloss.Left, progressive, m.list.View())
}

func listPaths(roots []string) {
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			for _, ext := range []string{".mp3", ".wav", ".oga", ".ogg", ".spx", ".opus", ".flac"} {
				if filepath.Ext(d.Name()) == ext {
					fileList = append(fileList, audiofile{
						filename: path,
					})
					break
				}
			}
			return nil
		})
	}
}

func main() {
	config := LoadConfig()

	listPaths(config.Directories)
	var trackList []list.Item

	noOfFiles := len(fileList)

	for i := 0; i < noOfFiles; i++ {
		file, err := os.Open(fileList[i].filename)
		if err != nil {
			log.Fatal(err)
		}

		tr, err := tag.ReadFrom(file)
		if err != nil {
			file.Close()
			continue
		}

		var audioformat musicType
		switch tr.FileType() {
		case "MP3":
			audioformat = typeMP3
		case "OGG":
			audioformat = typeOgg
		case "FLAC":
			audioformat = typeFlac
		default:
			audioformat = typeWav
		}

		fileList[i].file, fileList[i].audioformat = file, audioformat
		trackList = append(trackList, track{
			title:       tr.Title(),
			description: tr.Album() + "⋅" + tr.Artist(),
		})
	}

	m := newModel(trackList)
	t := tea.NewProgram(*m, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := t.Run(); err != nil {
		log.Fatal(err)
	}

	if player != nil {
		player.Close()
	}

	for i := 0; i < noOfFiles; i++ {
		fileList[i].file.Close()
	}
}
